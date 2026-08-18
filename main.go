package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/snuskin-1313/gator/internal/config"
	"github.com/snuskin-1313/gator/internal/database"
	"github.com/snuskin-1313/gator/internal/feeds"
)

type state struct {
	db     *database.Queries
	config *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	f, exists := c.cmds[cmd.name]
	if exists {
		err := f(s, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 || len(cmd.args) > 1 {
		return fmt.Errorf("please provide a username")
	}
	ctx := context.Background()
	if _, err := s.db.GetUser(ctx, cmd.args[0]); err != nil {
		fmt.Printf("err: %s\n", err)
		os.Exit(1)
	}
	s.config.CurrentUserName = cmd.args[0]
	s.config.SetUser(s.config.CurrentUserName)
	fmt.Println("Username has been set")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 || len(cmd.args) > 1 {
		return fmt.Errorf("please provide a username")
	}

	ctx := context.Background()
	if u, _ := s.db.GetUser(ctx, cmd.args[0]); u.Name == cmd.args[0] {
		fmt.Printf("user %s already exists\n", cmd.args[0])
		os.Exit(1)
	}

	newID := uuid.New()
	userTime := time.Now()

	_, err := s.db.CreateUser(ctx, database.CreateUserParams{
		ID:        int32(newID.ID()),
		CreatedAt: userTime.UTC(),
		UpdatedAt: userTime.UTC(),
		Name:      cmd.args[0],
	})
	if err != nil {
		return err
	}

	s.config.SetUser(cmd.args[0])
	fmt.Printf("user %s created\n", cmd.args[0])
	return nil
}

func handlerReset(s *state, cmd command) error {
	ctx := context.Background()
	err := s.db.DeleteUsers(ctx)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Successful reset")
	os.Exit(0)
	return nil
}

func handlerUsers(s *state, cmd command) error {
	ctx := context.Background()

	users, err := s.db.GetUsers(ctx)
	if err != nil {
		return err
	}

	for _, user := range users {
		if s.config.CurrentUserName == user.Name {
			fmt.Printf("* %s (current)\n", user.Name)
			continue
		}
		fmt.Printf("* %s\n", user.Name)
	}

	return nil
}

func scrapeFeeds(ctx context.Context, s *state) error {
	nextfeed, err := s.db.GetNextFeedToFetch(ctx)
	if err != nil {
		return err
	}

	now := sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}

	params := database.MarkFeedFetchedParams{
		ID:            nextfeed.ID,
		LastFetchedAt: now,
	}

	s.db.MarkFeedFetched(ctx, params)
	feed, err := fetchFeed(ctx, nextfeed.Url)
	if err != nil {
		return err
	}

	for _, i := range feed.Channel.Item {
		postID := uuid.New()
		// pubDate, err := time.Parse(i.PubDate)
		if err != nil {
			return err
		}
		fmt.Printf("i.pubDate:\n%s\n", i.PubDate)
		postParams := database.CreatePostParams{
			ID:          int32(postID.ID()),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       i.Title,
			Url:         i.Link,
			Description: i.Description,
			PublishedAt: time.Now().UTC(),
			FeedID:      nextfeed.ID,
		}

		s.db.CreatePost(ctx, postParams)
	}
	return nil
}

// arg[0]: time_between_reqs
func handlerAgg(s *state, cmd command) error {
	ctx := context.Background()
	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	ticker := time.NewTicker(timeBetweenReqs)
	fmt.Printf("Scraping for every %s\n", timeBetweenReqs)
	for ; ; <-ticker.C {
		fmt.Println("Scraping now")
		scrapeFeeds(ctx, s)
	}

	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*feeds.RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return &feeds.RSSFeed{}, err
	}
	req.Header.Set("User-Agent", "gator")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := feeds.RSSFeed{}

	fmt.Println(string(data[0:100]))
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for j := range feed.Channel.Item {
		feed.Channel.Item[j].Title = html.UnescapeString(feed.Channel.Item[j].Title)
		feed.Channel.Item[j].Description = html.UnescapeString(feed.Channel.Item[j].Description)
	}

	return &feed, nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	newID := uuid.New()
	userTime := time.Now()

	params := database.CreateFeedParams{
		ID:        int32(newID.ID()),
		CreatedAt: userTime.UTC(),
		UpdatedAt: userTime.UTC(),
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	}
	_, err2 := s.db.CreateFeed(ctx, params)
	if err2 != nil {
		return err2
	}

	paramsFF := database.CreateFeedFollowsParams{
		ID:        int32(newID.ID()),
		CreatedAt: userTime.UTC(),
		UpdatedAt: userTime.UTC(),
		UserID:    user.ID,
		FeedID:    int32(newID.ID()),
	}
	_, err2 = s.db.CreateFeedFollows(ctx, paramsFF)
	if err2 != nil {
		return err2
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	ctx := context.Background()
	data, err := s.db.GetFeeds(ctx)
	if err != nil {
		return err
	}

	for _, row := range data {
		id, err := s.db.GetUserFromID(ctx, row.UserID)
		if err != nil {
			return err
		}
		fmt.Println(row.Name, row.Url, id)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	newID := uuid.New()
	userTime := time.Now()

	gotFeed, err := s.db.GetFeedFromUrl(ctx, cmd.args[0])
	if err != nil {
		return err
	}
	feedID := gotFeed[0].ID

	params := database.CreateFeedFollowsParams{
		ID:        int32(newID.ID()),
		CreatedAt: userTime.UTC(),
		UpdatedAt: userTime.UTC(),
		UserID:    user.ID,
		FeedID:    feedID,
	}

	_, err2 := s.db.CreateFeedFollows(ctx, params)
	if err2 != nil {
		return err
	}

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	ctx := context.Background()

	userFeeds, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return err
	}

	fmt.Printf("User %s Following:\n", s.config.CurrentUserName)
	for _, f := range userFeeds {
		fmt.Printf("%s\n", f.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	feeds, err := s.db.GetFeedFromUrl(ctx, cmd.args[0])
	if err != nil {
		return err
	}
	feedID := feeds[0].ID
	params := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feedID,
	}

	if err := s.db.DeleteFeedFollow(ctx, params); err != nil {
		return err
	}

	return nil
}

func handlerBrowse(s *state, cmd command) error {
	ctx := context.Background()
	var limit int
	var err error
	if len(cmd.args) == 0 {
		limit = 2
	} else {
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
	}
	posts, err2 := s.db.GetPostsForUser(ctx, int32(limit))
	if err2 != nil {
		return err2
	}

	for _, p := range posts {
		fmt.Printf("%s | %s | %s\n%s\n", p.Title, p.Url, p.PublishedAt, p.Description)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		ctx := context.Background()
		curUser, err := s.db.GetUser(ctx, s.config.CurrentUserName)
		if err != nil {
			return err
		}

		return handler(s, cmd, curUser)
	}
}

func main() {
	curState := state{}
	c := config.Read()
	curState.config = &c

	db, err := sql.Open("postgres", curState.config.DBURL)
	if err != nil {
		fmt.Printf("error: %s \n", err)
		os.Exit(69)
	}
	dbQueries := database.New(db)

	curState.db = dbQueries

	cmds := commands{}
	cmds.cmds = make(map[string]func(*state, command) error)
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", handlerBrowse)

	if len(os.Args) < 2 {
		fmt.Printf("illegal arguments\n")
		os.Exit(1)
	}

	newCmd := command{
		name: os.Args[1],
		args: os.Args[2:],
	}

	if err := cmds.run(&curState, newCmd); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

# GATOR

An RSS aggregator written in Go.

Requirements:
- Go
- PostgreSQL

## INSTALLATION

Install by running
`go install gator`

Place **.gatorconfig.json** in your home directory as 
`{"db_url":"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable","current_user_name":null}`
You can then run gator as
`./gator <command>`

## COMMANDS

### register
Register a new user
`register <username>`

### login
Login for a given username
`login <username>`

### reset
Delete users and all corresponding data
`reset`

### users
View information about all users
`users`

### addfeed
Add an RSS feed. Requires a currently logged in user
`addfeed <feed name> <url>`

### feeds
Display all feeds stored in Gator
`feeds`

### follow
Follow a feed from a given url. Requires feed to be added first
`follow <url>`

### following
Display all the feeds the current user is following
`following`

### unfollow
Unfollow a certain feed from a URL
`unfollow <url>`

### agg
Grab feeds on a set interval (format: "0h0m0s", ex. "5m30s")
`agg <interval>`

### browse
Get all of the posts from the user's feeds and display in the console
`browse [limit]`

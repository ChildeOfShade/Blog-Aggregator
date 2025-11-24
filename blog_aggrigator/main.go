package main

import (
    "context"
    "database/sql"
    "encoding/xml"
    "fmt"
    "html"
    "io"
    "log"
    "net/http"
    "os"
    "time"

    _ "github.com/lib/pq"
    "github.com/google/uuid"

    "gator/internal/config"
    "gator/internal/database"
)


type state struct {
    db  *database.Queries
    cfg *config.Config
}

type command struct {
    name string
    args []string
}

type commands struct {
    handlers map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
    c.handlers[name] = f
}

func (c *commands) run(s *state, cmd command) error {
    handler, ok := c.handlers[cmd.name]
    if !ok {
        return fmt.Errorf("unknown command: %s", cmd.name)
    }
    return handler(s, cmd)
}

//
// LOGIN HANDLER
//
func handlerLogin(s *state, cmd command) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("username required")
    }
    username := cmd.args[0]

    // Check if user exists
    user, err := s.db.GetUser(context.Background(), username)
    if err != nil {
        return fmt.Errorf("user does not exist: %s", username)
    }

    // Save to config
    if err := s.cfg.SetUser(user.Name); err != nil {
        return err
    }

    fmt.Println("logged in as", user.Name)
    return nil
}

//
// REGISTER HANDLER
//
func handlerRegister(s *state, cmd command) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("username required")
    }
    username := cmd.args[0]

    // Check if user already exists
    _, err := s.db.GetUser(context.Background(), username)
    if err == nil {
        fmt.Println("user already exists:", username)
        os.Exit(1)
    }

    // Create new user
    newUser := database.CreateUserParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Name:      username,
    }

    user, err := s.db.CreateUser(context.Background(), newUser)
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    // Save user to config
    if err := s.cfg.SetUser(user.Name); err != nil {
        return err
    }

    fmt.Println("user created:", user.Name)
    fmt.Printf("%+v\n", user) // debug logging
    return nil
}

func main() {
    // Load config
    cfg, err := config.Read()
    if err != nil {
        log.Fatal(err)
    }

    // Connect to database (fix field name here)
    db, err := sql.Open("postgres", cfg.DBUrl)
    if err != nil {
        log.Fatal(err)
    }

    dbQueries := database.New(db)

    s := &state{
        db:  dbQueries,
        cfg: &cfg,
    }

    cmds := commands{
        handlers: make(map[string]func(*state, command) error),
    }

    // Register our commands
    cmds.register("login", handlerLogin)
    cmds.register("register", handlerRegister)
    cmds.register("reset", handlerReset)
    cmds.register("users", handlerUsers)
    cmds.register("agg", handlerAgg)
    cmds.register("addfeed", handlerAddFeed)


    // Parse CLI args
    if len(os.Args) < 2 {
        fmt.Println("not enough arguments")
        os.Exit(1)
    }

    cmd := command{
        name: os.Args[1],
        args: os.Args[2:],
    }

    if err := cmds.run(s, cmd); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

//
// RESET HANDLER
//
func handlerReset(s *state, cmd command) error {
    err := s.db.ResetUsers(context.Background())
    if err != nil {
        fmt.Println("failed to reset database:", err)
        return err
    }
    fmt.Println("database successfully reset")
    return nil
}

//
// USERS HANDLER
//
func handlerUsers(s *state, cmd command) error {
    users, err := s.db.GetUsers(context.Background())
    if err != nil {
        return fmt.Errorf("failed to list users: %w", err)
    }

    current := s.cfg.CurrentUserName

    for _, u := range users {
        if u.Name == current {
            fmt.Printf("* %s (current)\n", u.Name)
        } else {
            fmt.Printf("* %s\n", u.Name)
        }
    }

    return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	// Unescape top-level fields
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	// Unescape each item
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
	}

	return &feed, nil
}

func handlerAgg(s *state, cmd command) error {
    ctx := context.Background()

    feed, err := fetchFeed(ctx, "https://www.wagslane.dev/index.xml")
    if err != nil {
        return err
    }

    // Print entire struct (Boot.dev expects this)
    fmt.Printf("%+v\n", feed)
    return nil
}

func handlerAddFeed(s *state, cmd command) error {
    if len(cmd.args) != 2 {
        return fmt.Errorf("usage: addfeed <name> <url>")
    }

    name := cmd.args[0]
    url := cmd.args[1]

    // Get current user
    user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
    if err != nil {
        return fmt.Errorf("must be logged in: %w", err)
    }

    feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
        ID:     uuid.New(),
        Name:   name,
        Url:    url,
        UserID: user.ID,
    })
    if err != nil {
        return err
    }

    fmt.Printf("Feed created:\n")
    fmt.Printf("ID: %s\n", feed.ID)
    fmt.Printf("Name: %s\n", feed.Name)
    fmt.Printf("URL: %s\n", feed.Url)
    fmt.Printf("UserID: %s\n", feed.UserID)

    return nil
}

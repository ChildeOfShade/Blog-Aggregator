package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
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

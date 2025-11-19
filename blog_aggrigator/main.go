package main

import (
    "fmt"
    "log"
    "os"

    "gator/internal/config"
)

type state struct {
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

func handlerLogin(s *state, cmd command) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("username required")
    }
    username := cmd.args[0]

    if err := s.cfg.SetUser(username); err != nil {
        return err
    }

    fmt.Println("current user set to", username)
    return nil
}

func main() {
    // 1. Read config file
    cfg, err := config.Read()
    if err != nil {
        log.Fatal(err)
    }

    // 2. Create state
    s := &state{cfg: &cfg}

    // 3. Create commands registry
    cmds := commands{
        handlers: make(map[string]func(*state, command) error),
    }

    // 4. Register login handler
    cmds.register("login", handlerLogin)

    // 5. Parse CLI args
    if len(os.Args) < 2 {
        fmt.Println("not enough arguments")
        os.Exit(1)
    }

    name := os.Args[1]
    args := os.Args[2:]

    cmd := command{name: name, args: args}

    // 6. Run the command
    if err := cmds.run(s, cmd); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

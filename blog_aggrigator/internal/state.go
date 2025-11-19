type state struct {
    cfg *config.Config
}

type command struct {
    name string
    args []string
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

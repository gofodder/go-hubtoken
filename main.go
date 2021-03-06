package main

import (
  "os"
  "fmt"
  "bufio"
  "syscall"
  "strings"
  "golang.org/x/crypto/ssh/terminal"
  "github.com/codegangsta/cli"
  "github.com/google/go-github/github"
  "github.com/fatih/color"
)

func WarningMessage() *color.Color {
  return color.New(color.FgYellow, color.Bold)
}

func InfoMessage() *color.Color {
  return color.New(color.FgGreen, color.Bold)
}

func SuccessMessage() *color.Color {
  return color.New(color.FgCyan, color.Bold)
}

func HeadingMessage() *color.Color {
  return color.New(color.Bold)
}

func ErrorMessage() *color.Color {
  return color.New(color.FgRed, color.Bold)
}

func isPersonalAccessToken(a *github.Authorization) bool {
  return *a.App.URL == "https://developer.github.com/v3/oauth_authorizations/"
}

type Authorizations []*github.Authorization

func (authorizations Authorizations) Filter(f func(*github.Authorization) bool) Authorizations {
  filtered := make(Authorizations, 0)
  for _, authorization := range authorizations {
    if isPersonalAccessToken(authorization) && f(authorization) {
      filtered = append(filtered, authorization)
    }
  }
  return filtered
}

func (authorizations Authorizations) ForEach(f func(*github.Authorization)) {
  for _, authorization := range authorizations {
    if isPersonalAccessToken(authorization) {
      f(authorization)
    }
  }
}

func GetAuthorization(note string, authorizations Authorizations) *github.Authorization {
  if len(authorizations) < 1 {
    WarningMessage().Printf("Warning:")
    fmt.Println("There are no personal access tokens for your GitHub account")
    return nil
  }

  filtered := authorizations.Filter(func(auth *github.Authorization) bool {
    return *auth.Note == note
  })

  if len(filtered) < 1 {
    WarningMessage().Printf("Warning:")
    fmt.Printf("There are no personal access tokens for your GitHub account named %s\n", note)
  }

  if len(filtered) == 1 {
    return filtered[0]
  }

  ErrorMessage().Printf("Error:")
  fmt.Printf("Unhandled error when getting token")
  return nil
}

func Prompt(Message string) string {
  fmt.Print(Message)
  reader := bufio.NewReader(os.Stdin)
  text, _ := reader.ReadString('\n')
  return strings.TrimSpace(text)
}

func PasswordPrompt(Message string) string {
  fmt.Print(Message)
  bytes, _ := terminal.ReadPassword(int(syscall.Stdin))
  text := string(bytes)
  fmt.Println()
  return strings.TrimSpace(text)
}

func Login() *github.Client {
  transport := github.BasicAuthTransport{
    Username: Prompt("Github login: "),
    Password: PasswordPrompt("Password: "),
    OTP:      Prompt("2FA/OTP: "),
  }
  fmt.Println()
  return github.NewClient(transport.Client())
}

func CreateToken(note string) {
  client := Login()

  // TODO: Scopes should be set by the user
  scopes := []github.Scope{"repo"}

  auth_req := &github.AuthorizationRequest{
    Note: &note,
    Scopes: scopes,
  }

  authorization, _, err := client.Authorizations.Create(auth_req)

  if err != nil {
    fmt.Printf("Error creating personal access token: %v\nAuthorizations.Create returned error: %v\n", note, err)
  } else {
    token := *authorization.Token
    fmt.Printf("%v\n", strings.TrimPrefix(token, "0x"))
  }
}

func GetAuthorizationsList(client *github.Client) Authorizations {
  authorizations, _, err := client.Authorizations.List(nil)
  if err != nil {
    ErrorMessage().Printf("Error getting personal access tokens\nAuthorizations.List returned error: %v\n", err)
    os.Exit(1)
  }
  return authorizations
}

func DeleteToken(note string) {
  client := Login()
  authorization := GetAuthorization(note, GetAuthorizationsList(client))

  if authorization != nil {
    _, err := client.Authorizations.Delete(*authorization.ID)
    if err != nil {
      WarningMessage().Printf("Error deleting personal access token: %v\n", note)
      ErrorMessage().Printf("Authorizations.Delete returned error: %v\n", err)
    } else {
      SuccessMessage().Printf("Deleted personal access token %v\n", note)
    }
  } else {
    WarningMessage().Printf("Couldn't find personal access token %v\n", note)
  }
}

func ListTokens() {
  client := Login()
  authorizations := GetAuthorizationsList(client)
  if len(authorizations) > 0 {
    HeadingMessage().Printf("GitHub Personal Access Tokens:\n")
    authorizations.ForEach(func(auth *github.Authorization) {
      InfoMessage().Printf("%s\n", *auth.Note)
    })
  } else {
    ErrorMessage().Printf("There are no personal access tokens for your GitHub account\n")
  }
}

func main() {
  app           := cli.NewApp()
  app.Name       = "HubToken"
  app.Usage      = "Manage GitHub personal access tokens\n" +
    "(use a github personal token in-place of username or password when prompted to skip OTP/2FA)"
  app.Version    = "1.0.4"
  app.Commands   = []cli.Command{
    {
      Name: "create",
      Aliases: []string{"c"},
      Usage: "Create personal access token called `NAME`",
      Action: func(c *cli.Context) error {
        if c.Args().First() != "" {
          CreateToken(c.Args().First())
        } else {
          WarningMessage().Printf("You must supply a token name to create.")
          os.Exit(1)
        }
        return nil
      },
    },
    {
      Name: "delete",
      Aliases: []string{"D"},
      Usage: "Delete personal access token called `NAME`",
      Action: func(c *cli.Context) error {
        if c.Args().First() != "" {
          DeleteToken(c.Args().First())
        } else {
          WarningMessage().Printf("You must supply a token name to delete.")
          os.Exit(1)
        }
        return nil
      },
    },
    {
      Name: "list",
      Aliases: []string{"l"},
      Usage: "List all personal access tokens",
      Action: func(c *cli.Context) error {
        ListTokens()
        return nil
      },
    },
  }

  app.Run(os.Args)
}

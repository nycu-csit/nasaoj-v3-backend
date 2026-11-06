package utils

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// ReadPassword prompts for a password on stderr. On a terminal the input is
// not echoed; otherwise the first line of stdin is used, so scripts can pipe
// the password in.
func ReadPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// LoginPlain logs in with username/password via POST /auth/login/plain,
// prompting for the password. The session cookies the backend sets are
// added to client and returned, so callers can persist them.
func LoginPlain(
	ctx context.Context,
	client *resty.Client,
	username string,
) ([]*http.Cookie, error) {
	password, err := ReadPassword(
		fmt.Sprintf("Password for %s: ", username))
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	loginURL, err := url.JoinPath(
		viper.GetString("endpoint"), "/auth/login/plain")
	if err != nil {
		return nil, err
	}

	// A wrong password is not worth retrying: use a fresh request without
	// the client's retry condition.
	resp, err := resty.New().R().
		SetContext(ctx).
		SetBody(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(loginURL)
	if err != nil {
		return nil, fmt.Errorf("POST /auth/login/plain failed: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("login failed (%s): %s",
			resp.Status(), strings.TrimSpace(string(resp.Body())))
	}

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		client.SetCookie(cookie)
	}
	return cookies, nil
}

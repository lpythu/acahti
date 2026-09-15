package catalog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"acahti/internal/page"
	"acahti/internal/woodpecker"
)

var secretName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func NormalizeSecretName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !secretName.MatchString(name) {
		return "", fmt.Errorf("%w: secret name", ErrInvalid)
	}
	return name, nil
}

func secretEvents(events []string) []string {
	if len(events) == 0 {
		return append([]string{}, woodpecker.DefaultSecretEvents...)
	}
	out := make([]string, 0, len(events))
	for _, e := range events {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return append([]string{}, woodpecker.DefaultSecretEvents...)
	}
	return out
}

func (c *Catalog) requireOrgSecrets(user string) error {
	if c.IsOrgAdmin(user) {
		return nil
	}
	return ErrNotFound
}

func (c *Catalog) requireRepoSecrets(user, owner, name string) error {
	_, err := c.requireRepoAdmin(user, owner, name)
	if errors.Is(err, ErrForbidden) {
		return ErrNotFound
	}
	return err
}

func (c *Catalog) kernel() (*woodpecker.Client, error) {
	if c == nil || c.WP == nil || !c.WP.Ready() {
		return nil, fmt.Errorf("pipeline kernel unavailable")
	}
	return c.WP, nil
}

func (c *Catalog) ListOrgSecrets(user string, q page.Query) (page.Result[woodpecker.Secret], error) {
	if err := c.requireOrgSecrets(user); err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	wp, err := c.kernel()
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	return wp.ListOrgSecrets(c.Cfg.Org, q)
}

func (c *Catalog) ListRepoSecrets(user, owner, name string, q page.Query) (page.Result[woodpecker.Secret], error) {
	if err := c.requireRepoSecrets(user, owner, name); err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	wp, err := c.kernel()
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	return wp.ListRepoSecrets(owner+"/"+name, q)
}

func (c *Catalog) PutOrgSecret(user, name, value string, events []string) (woodpecker.Secret, error) {
	if err := c.requireOrgSecrets(user); err != nil {
		return woodpecker.Secret{}, err
	}
	name, err := NormalizeSecretName(name)
	if err != nil {
		return woodpecker.Secret{}, err
	}
	if strings.TrimSpace(value) == "" {
		return woodpecker.Secret{}, fmt.Errorf("%w: value required", ErrInvalid)
	}
	wp, err := c.kernel()
	if err != nil {
		return woodpecker.Secret{}, err
	}
	return wp.PutOrgSecret(c.Cfg.Org, name, value, secretEvents(events))
}

func (c *Catalog) PutRepoSecret(user, owner, name, secret, value string, events []string) (woodpecker.Secret, error) {
	if err := c.requireRepoSecrets(user, owner, name); err != nil {
		return woodpecker.Secret{}, err
	}
	secret, err := NormalizeSecretName(secret)
	if err != nil {
		return woodpecker.Secret{}, err
	}
	if strings.TrimSpace(value) == "" {
		return woodpecker.Secret{}, fmt.Errorf("%w: value required", ErrInvalid)
	}
	wp, err := c.kernel()
	if err != nil {
		return woodpecker.Secret{}, err
	}
	return wp.PutRepoSecret(owner+"/"+name, secret, value, secretEvents(events))
}

func (c *Catalog) DeleteOrgSecret(user, name string) error {
	if err := c.requireOrgSecrets(user); err != nil {
		return err
	}
	name, err := NormalizeSecretName(name)
	if err != nil {
		return err
	}
	wp, err := c.kernel()
	if err != nil {
		return err
	}
	return wp.DeleteOrgSecret(c.Cfg.Org, name)
}

func (c *Catalog) DeleteRepoSecret(user, owner, name, secret string) error {
	if err := c.requireRepoSecrets(user, owner, name); err != nil {
		return err
	}
	secret, err := NormalizeSecretName(secret)
	if err != nil {
		return err
	}
	wp, err := c.kernel()
	if err != nil {
		return err
	}
	return wp.DeleteRepoSecret(owner+"/"+name, secret)
}

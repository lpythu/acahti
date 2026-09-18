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
	if c == nil || c.wp == nil || !c.wp.Ready() {
		return nil, fmt.Errorf("pipeline kernel unavailable")
	}
	return c.wp, nil
}

func (c *Catalog) ListOrgSecrets(user string, q page.Query) (page.Result[woodpecker.Secret], error) {
	if err := c.requireOrgSecrets(user); err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	wp, err := c.kernel()
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	listed, err := wp.ListOrgSecrets(c.Cfg.Org, q)
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	for i := range listed.Items {
		listed.Items[i].Scope = "org"
	}
	return listed, nil
}

func (c *Catalog) ListRepoSecrets(user, owner, name string, q page.Query) (page.Result[woodpecker.Secret], error) {
	if err := c.requireRepoSecrets(user, owner, name); err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	wp, err := c.kernel()
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	repo, err := wp.ListRepoSecrets(owner+"/"+name, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	org, err := wp.ListOrgSecrets(c.Cfg.Org, page.Query{Page: 1, Size: page.MaxSize})
	if err != nil {
		return page.Result[woodpecker.Secret]{}, err
	}
	return page.Clip(mergeEffectiveSecrets(org.Items, repo.Items), q), nil
}

func mergeEffectiveSecrets(org, repo []woodpecker.Secret) []woodpecker.Secret {
	seen := map[string]struct{}{}
	out := make([]woodpecker.Secret, 0, len(org)+len(repo))
	for _, s := range repo {
		s.Scope = "repo"
		out = append(out, s)
		seen[s.Name] = struct{}{}
	}
	for _, s := range org {
		if _, ok := seen[s.Name]; ok {
			continue
		}
		s.Scope = "org"
		out = append(out, s)
	}
	return out
}

func (c *Catalog) PutOrgSecret(user, name, value string) (woodpecker.Secret, error) {
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
	return wp.PutOrgSecret(c.Cfg.Org, name, value)
}

func (c *Catalog) PutRepoSecret(user, owner, name, secret, value string) (woodpecker.Secret, error) {
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
	return wp.PutRepoSecret(owner+"/"+name, secret, value)
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

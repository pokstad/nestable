package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/pokstad/nestable/orm"
)

type getConfigCmd struct {
	repo orm.Repo
	key  *string
}

func newGetConfigCmd(repo orm.Repo) subCmd {
	return &getConfigCmd{repo: repo}
}

func (_ *getConfigCmd) Help() string {
	return `Get the config value for a given key.`
}

func (_ *getConfigCmd) Names() []string {
	return []string{"get-config", "gc"}
}

func (gcc *getConfigCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("set-config", flag.ExitOnError)
	gcc.key = fs.String("key", "", "config key to get")
	return fs
}

func (gcc *getConfigCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	if *gcc.key == "" {
		keys, err := gcc.repo.GetConfigKeys(ctx)
		if err != nil {
			return fmt.Errorf("getting config keys: %w", err)
		}
		i, err := fuzzyfinder.Find(keys, func(i int) string {
			return keys[i]
		})
		if err != nil {
			return fmt.Errorf("selecting config key: %w", err)
		}
		*gcc.key = keys[i]
	}

	value, err := gcc.repo.GetConfig(ctx, *gcc.key)
	if err != nil {
		return fmt.Errorf("getting config %q: %w", *gcc.key, err)
	}
	_, err = fmt.Fprintln(w, value)
	return err
}

type setConfigCmd struct {
	repo  orm.Repo
	key   *string
	value *string
}

func newSetConfigCmd(repo orm.Repo) subCmd {
	return &setConfigCmd{repo: repo}
}

func (_ *setConfigCmd) Help() string {
	return `Set the config value for a given key.`
}

func (_ *setConfigCmd) Names() []string {
	return []string{"set-config", "sc"}
}

func (gcc *setConfigCmd) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("set-config", flag.ExitOnError)
	gcc.key = fs.String("key", "", "config key to change")
	gcc.value = fs.String("value", "", "config value to change to")
	return fs
}

func (gcc *setConfigCmd) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	err := gcc.repo.SetConfig(ctx, *gcc.key, *gcc.value)
	if err != nil {
		return fmt.Errorf("setting config %q: %w", *gcc.key, err)
	}
	return nil
}

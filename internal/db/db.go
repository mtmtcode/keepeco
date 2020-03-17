package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/tobischo/gokeepasslib/v3"
)

type Database struct {
	*gokeepasslib.Database
}

// Open returns KeepassXC database
func Open(path, password string) (*Database, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(password)
	err = gokeepasslib.NewDecoder(f).Decode(db)
	if err != nil {
		return nil, err
	}
	return &Database{db}, nil
}

// Flatten returns a slice of all entries in the database
func (d *Database) Flatten() Entries {
	entries := make([]Entry, 0, 20)

	var flattenGroup func(string, *gokeepasslib.Group)
	flattenGroup = func(prefix string, grp *gokeepasslib.Group) {
		for i := range grp.Entries {
			entry := Entry{prefix, &grp.Entries[i]}
			entries = append(entries, entry)
		}

		for _, subgrp := range grp.Groups {
			if prefix == "" && (strings.HasPrefix(subgrp.Name, "99_trash") || subgrp.Name == "ゴミ箱" || subgrp.Name == "Backup") {
				continue
			}
			subPrefix := fmt.Sprintf("%s/%s", prefix, strings.ReplaceAll(subgrp.Name, "\n", ""))
			flattenGroup(subPrefix, &subgrp)
		}
	}
	flattenGroup("", &d.Content.Root.Groups[0])
	return Entries(entries)
}

type Entry struct {
	Prefix string
	*gokeepasslib.Entry
}

type Entries []Entry

// Candidates returns a slice of string expressions of the Entries
func (es Entries) Candidates() []string {
	r := make([]string, len(es))
	for i, e := range es {
		r[i] = fmt.Sprintf("%s/%s", e.Prefix, strings.ReplaceAll(e.GetContent("Title"), "\n", ""))
	}
	return r
}

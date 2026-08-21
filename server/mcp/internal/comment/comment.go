// Package comment 是 Inbox / Proposal 共用的评论对象（SCHEMA §12.4）。
//
// 只追加，不改、不删。形状只在这里实现一份。
package comment

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Comment struct {
	ID         string    `json:"id" yaml:"id"`
	AuthorType string    `json:"author_type" yaml:"author_type"`
	Author     string    `json:"author" yaml:"author"`
	At         time.Time `json:"at" yaml:"at"`
	Body       string    `json:"body" yaml:"body"`
	ReplyTo    string    `json:"reply_to,omitempty" yaml:"reply_to,omitempty"`
}

type Input struct {
	Body    string `json:"body"`
	ReplyTo string `json:"reply_to,omitempty"`
}

func New(authorType, author string, in Input) (Comment, error) {
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return Comment{}, errors.New("required field comment.body missing")
	}
	switch authorType {
	case "human", "agent":
	default:
		authorType = "human"
	}
	if author == "" {
		author = "unknown"
	}
	now := time.Now()
	id := "cmt_" + now.Format("20060102_150405") + fmt.Sprintf("_%03d", now.Nanosecond()/1e6)
	return Comment{
		ID:         id,
		AuthorType: authorType,
		Author:     author,
		At:         now,
		Body:       body,
		ReplyTo:    strings.TrimSpace(in.ReplyTo),
	}, nil
}

func Slice(cs []Comment) []Comment {
	if cs == nil {
		return []Comment{}
	}
	return cs
}

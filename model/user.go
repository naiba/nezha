package model

import (
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/google/go-github/v47/github"
	"github.com/xanzy/go-gitlab"
)

type User struct {
	Common
	Login     string `json:"login,omitempty"`      // 登录名
	AvatarURL string `json:"avatar_url,omitempty"` // 头像地址
	Name      string `json:"name,omitempty"`       // 昵称
	Blog      string `json:"blog,omitempty"`       // 网站链接
	Email     string `json:"email,omitempty"`      // 邮箱
	Hireable  bool   `json:"hireable,omitempty"`
	Bio       string `json:"bio,omitempty"` // 个人简介

	Token        string    `json:"-"`                       // 认证 Token
	TokenExpired time.Time `json:"token_expired,omitempty"` // Token 过期时间
	SuperAdmin   bool      `json:"super_admin,omitempty"`   // 超级管理员
}

func NewUserFromGitea(gu *gitea.User) User {
	var u User
	u.ID = uint64(gu.ID)
	u.Login = gu.UserName
	u.AvatarURL = gu.AvatarURL
	u.Name = gu.FullName
	if u.Name == "" {
		u.Name = u.Login
	}
	u.Blog = gu.Website
	u.Email = gu.Email
	u.Bio = gu.Description
	return u
}

func NewUserFromGitlab(gu *gitlab.User) User {
	var u User
	u.ID = uint64(gu.ID)
	u.Login = gu.Username
	u.AvatarURL = gu.AvatarURL
	u.Name = gu.Name
	if u.Name == "" {
		u.Name = u.Login
	}
	u.Blog = gu.WebsiteURL
	u.Email = gu.Email
	u.Bio = gu.Bio
	return u
}

func NewUserFromGitHub(gu *github.User) User {
	var u User
	u.ID = uint64(gu.GetID())
	u.Login = gu.GetLogin()
	u.AvatarURL = gu.GetAvatarURL()
	u.Name = gu.GetName()
	// 昵称为空的情况
	if u.Name == "" {
		u.Name = u.Login
	}
	u.Blog = gu.GetBlog()
	u.Email = gu.GetEmail()
	u.Hireable = gu.GetHireable()
	u.Bio = gu.GetBio()
	return u
}

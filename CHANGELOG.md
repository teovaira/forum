# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- SQLite schema: `users`, `sessions`, `posts`, `comments`, `categories`, `post_categories`, `reactions`,
  seeded with a fixed anime-themed category set.
- Registration and login with bcrypt-hashed passwords and UUID session tokens.
- Cookie-based session middleware; unauthenticated requests to protected routes redirect to `/login`.
- Creating posts with one or more categories; commenting on posts.
- Filtering the post feed by category, by the logged-in user's own posts, and by posts they've liked.
- Liking/disliking posts and comments, with a toggling reaction (like → dislike replaces it; repeating
  the same reaction removes it).
- Server-rendered HTML templates for the home feed, a single post, login, registration, the new-post
  form, and a generic error page.
- Progressive JavaScript enhancements: logout confirmation, relative timestamps, a post-body character
  counter — the site works fully with JavaScript disabled.
- End-to-end HTTP tests covering the full register → post → comment → react → logout journey.
- `Makefile` with build/run/test/lint/docker targets.
- Multi-stage `Dockerfile` for containerized builds.

### Fixed
- Session timestamps are now stored and compared in UTC everywhere, fixing sessions that could read
  as expired immediately on a non-UTC server.
- Foreign key enforcement now applies to every pooled database connection, not just the first.
- Post/comment body whitespace no longer gets extra indentation when rendered.

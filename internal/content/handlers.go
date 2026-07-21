package content

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/auth"
	"forum/internal/models"
	"forum/internal/webutil"
)

// HomePageData is the view-data contract for the home page (§4.4).
type HomePageData struct {
	Posts        []models.Post
	Categories   []models.Category
	CurrentUser  *models.User
	ActiveFilter string
}

// PostPageData is the view-data contract for the post details page (§4.4).
// It has been extended with an ErrorMessage to display comment-validation errors.
type PostPageData struct {
	Post         models.Post
	Comments     []models.Comment
	CurrentUser  *models.User
	ErrorMessage string
}

// NewPostPageData is the view-data contract for the new post creation page.
type NewPostPageData struct {
	Categories   []models.Category
	CurrentUser  *models.User
	ErrorMessage string
}

// HomeHandler serves the forum front page (GET /) and supports filters.
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - http.HandlerFunc: The handler.
func HomeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := auth.UserFromContext(r.Context())
		categoryParam := r.URL.Query().Get("category")
		filterParam := r.URL.Query().Get("filter")

		var filter PostFilter
		var activeFilter string

		if categoryParam != "" {
			catID, err := strconv.ParseInt(categoryParam, 10, 64)
			if err != nil {
				webutil.RenderError(w, http.StatusBadRequest, "Invalid category ID.")
				return
			}
			filter.CategoryID = &catID
			activeFilter = "category:" + categoryParam
		}

		if filterParam == "mine" {
			if currentUser == nil {
				webutil.RenderError(w, http.StatusUnauthorized, "You must be logged in to view your posts.")
				return
			}
			filter.CreatedByUserID = &currentUser.ID
			activeFilter = "mine"
		} else if filterParam == "liked" {
			if currentUser == nil {
				webutil.RenderError(w, http.StatusUnauthorized, "You must be logged in to view liked posts.")
				return
			}
			filter.LikedByUserID = &currentUser.ID
			activeFilter = "liked"
		}

		posts, err := ListPosts(db, filter)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load posts.")
			return
		}

		// Batch fetch categories and reaction counts for the loaded posts
		var postIDs []int64
		for _, p := range posts {
			postIDs = append(postIDs, p.ID)
		}

		if len(postIDs) > 0 {
			postCatMap, err := PostCategories(db, postIDs)
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories for posts.")
				return
			}

			reactionMap, err := CountReactions(db, models.TargetPost, postIDs)
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load reactions for posts.")
				return
			}

			for i := range posts {
				id := posts[i].ID
				posts[i].Categories = postCatMap[id]
				if counts, ok := reactionMap[id]; ok {
					posts[i].Likes = counts.Likes
					posts[i].Dislikes = counts.Dislikes
				}
			}
		}

		categories, err := ListCategories(db)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories.")
			return
		}

		data := HomePageData{
			Posts:        posts,
			Categories:   categories,
			CurrentUser:  currentUser,
			ActiveFilter: activeFilter,
		}

		if err := webutil.RenderTemplate(w, "home.html", data); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not render template.")
		}
	}
}

// PostViewHandler serves a single post detail page (GET /posts/{id}).
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - http.HandlerFunc: The handler.
func PostViewHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		postID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid post ID.")
			return
		}

		post, err := GetPost(db, postID)
		if err == sql.ErrNoRows {
			webutil.RenderError(w, http.StatusNotFound, "Post not found.")
			return
		} else if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load post.")
			return
		}

		// Load categories for the single post
		postCatMap, err := PostCategories(db, []int64{postID})
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories for post.")
			return
		}
		post.Categories = postCatMap[postID]

		// Load reaction counts for the single post
		reactionMap, err := CountReactions(db, models.TargetPost, []int64{postID})
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load reactions for post.")
			return
		}
		if counts, ok := reactionMap[postID]; ok {
			post.Likes = counts.Likes
			post.Dislikes = counts.Dislikes
		}

		comments, err := ListComments(db, postID)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load comments.")
			return
		}

		currentUser, _ := auth.UserFromContext(r.Context())

		data := PostPageData{
			Post:        *post,
			Comments:    comments,
			CurrentUser: currentUser,
		}

		if err := webutil.RenderTemplate(w, "post.html", data); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not render template.")
		}
	}
}

// NewPostFormHandler serves the creation form (GET /posts/new).
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - http.HandlerFunc: The handler.
func NewPostFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := auth.UserFromContext(r.Context())
		categories, err := ListCategories(db)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories.")
			return
		}

		data := NewPostPageData{
			Categories:  categories,
			CurrentUser: currentUser,
		}

		if err := webutil.RenderTemplate(w, "new_post.html", data); err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Could not render template.")
		}
	}
}

// CreatePostHandler handles submission of a new post (POST /posts).
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - http.HandlerFunc: The handler.
func CreatePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := auth.UserFromContext(r.Context())

		if err := r.ParseForm(); err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid form submission.")
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")
		categoryStrs := r.Form["categories"]

		var categoryIDs []int64
		for _, catStr := range categoryStrs {
			catID, err := strconv.ParseInt(catStr, 10, 64)
			if err != nil {
				webutil.RenderError(w, http.StatusBadRequest, "Invalid category ID selected.")
				return
			}
			categoryIDs = append(categoryIDs, catID)
		}

		// Re-render form with error message if validation fails
		reRenderFormWithError := func(errMsg string) {
			categories, err := ListCategories(db)
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories.")
				return
			}
			data := NewPostPageData{
				Categories:   categories,
				CurrentUser:  currentUser,
				ErrorMessage: errMsg,
			}
			if err := webutil.RenderTemplate(w, "new_post.html", data); err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not render template.")
			}
		}

		if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
			reRenderFormWithError("Title and body are required fields.")
			return
		}

		if len(categoryIDs) == 0 {
			reRenderFormWithError("At least one category must be selected.")
			return
		}

		postID, err := CreatePost(db, currentUser.ID, title, body, categoryIDs)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Failed to create post.")
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusSeeOther)
	}
}

// CreateCommentHandler handles submission of a comment (POST /posts/{id}/comments).
//
// Parameters:
//   - db: An open SQLite database connection.
//
// Returns:
//   - http.HandlerFunc: The handler.
func CreateCommentHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentUser, _ := auth.UserFromContext(r.Context())

		idStr := r.PathValue("id")
		postID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid post ID.")
			return
		}

		if err := r.ParseForm(); err != nil {
			webutil.RenderError(w, http.StatusBadRequest, "Invalid form submission.")
			return
		}

		body := r.FormValue("body")

		if strings.TrimSpace(body) == "" {
			// Re-render post page with validation error
			post, err := GetPost(db, postID)
			if err == sql.ErrNoRows {
				webutil.RenderError(w, http.StatusNotFound, "Post not found.")
				return
			} else if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load post.")
				return
			}

			// Load categories for the single post
			postCatMap, err := PostCategories(db, []int64{postID})
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load categories for post.")
				return
			}
			post.Categories = postCatMap[postID]

			// Load reaction counts for the single post
			reactionMap, err := CountReactions(db, models.TargetPost, []int64{postID})
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load reactions for post.")
				return
			}
			if counts, ok := reactionMap[postID]; ok {
				post.Likes = counts.Likes
				post.Dislikes = counts.Dislikes
			}

			comments, err := ListComments(db, postID)
			if err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not load comments.")
				return
			}

			data := PostPageData{
				Post:         *post,
				Comments:     comments,
				CurrentUser:  currentUser,
				ErrorMessage: "Comment body cannot be empty.",
			}

			if err := webutil.RenderTemplate(w, "post.html", data); err != nil {
				webutil.RenderError(w, http.StatusInternalServerError, "Could not render template.")
			}
			return
		}

		_, err = CreateComment(db, postID, currentUser.ID, body)
		if err != nil {
			webutil.RenderError(w, http.StatusInternalServerError, "Failed to post comment.")
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusSeeOther)
	}
}

// RegisterRoutes registers all content package HTTP routes on the multiplexer.
//
// Parameters:
//   - mux: The http.ServeMux instance.
//   - db: An open SQLite database connection.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /", HomeHandler(db))
	mux.HandleFunc("GET /posts/{id}", PostViewHandler(db))

	// Protected routes require authentication
	mux.Handle("GET /posts/new", auth.RequireAuth(http.HandlerFunc(NewPostFormHandler(db))))
	mux.Handle("POST /posts", auth.RequireAuth(http.HandlerFunc(CreatePostHandler(db))))
	mux.Handle("POST /posts/{id}/comments", auth.RequireAuth(http.HandlerFunc(CreateCommentHandler(db))))

	// Reaction routes owned by Vasiliki
	mux.Handle("POST /posts/{id}/like", auth.RequireAuth(ReactPostHandler(db)))
	mux.Handle("POST /posts/{id}/dislike", auth.RequireAuth(ReactPostHandler(db)))
	mux.Handle("POST /comments/{id}/like", auth.RequireAuth(ReactCommentHandler(db)))
	mux.Handle("POST /comments/{id}/dislike", auth.RequireAuth(ReactCommentHandler(db)))
}

package main

import (
	"errors"
	"fmt"
	"net/http"
	"snippetbox/internal/models"
	"snippetbox/internal/validator"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type snippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Tag                 string `form:"tag"`
	validator.Validator `form:"-"`
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

type commentaryForm struct {
	Author              map[string]string `form:"author"`
	Content             string            `form:"content"`
	validator.Validator `form:"-"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Snippets = snippets
	app.render(w, r, http.StatusOK, "home.html", data)
}
func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	idStr := params.ByName("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			fmt.Println(err)
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.Snippet = snippet
	data.Form = commentaryForm{}
	app.render(w, r, http.StatusOK, "view.html", data)
}
func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = snippetCreateForm{}
	app.render(w, r, http.StatusOK, "create.html", data)
}
func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form snippetCreateForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 100), "title", "This field cannot be more than 100 characters long")
	form.CheckField(validator.NotBlank(form.Tag), "tag", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "create.html", data)
		return
	}
	UserName := app.sessionManager.GetString(r.Context(), "UserName")
	UserIDStr := app.sessionManager.GetString(r.Context(), "authenticatedUserID")
	ObjectID, err := app.snippets.Insert(form.Title, form.Content, form.Tag, UserName, UserIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	id := ObjectID.Hex()
	app.sessionManager.Put(r.Context(), "flash", "Post successfully created!")
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%s", id), http.StatusSeeOther)
}

func (app *application) FavouritePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}
	params := httprouter.ParamsFromContext(r.Context())
	SnippetIDStr := params.ByName("id")
	SnippetID, err := primitive.ObjectIDFromHex(SnippetIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	UserIDStr := app.sessionManager.GetString(r.Context(), "authenticatedUserID")
	UserID, err := primitive.ObjectIDFromHex(UserIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	err = app.users.AddFavourites(SnippetID, UserID)
	if err != nil {
		if err.Error() == "Post is already in favourites" {
			app.sessionManager.Put(r.Context(), "flash", "Post is already in favourites!")
			http.Redirect(w, r, "/account/view", http.StatusSeeOther)
			return
		}
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Post added succesfuly!")
	http.Redirect(w, r, "/account/view", http.StatusSeeOther)
}
func (app *application) FavouriteDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.clientError(w, http.StatusMethodNotAllowed)
		return
	}
	params := httprouter.ParamsFromContext(r.Context())
	SnippetIDStr := params.ByName("id")
	SnippetID, err := primitive.ObjectIDFromHex(SnippetIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return

	}
	UserIDStr := app.sessionManager.GetString(r.Context(), "authenticatedUserID")
	UserID, err := primitive.ObjectIDFromHex(UserIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	Snippet, err := app.snippets.Get(SnippetID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	err = app.users.RemoveFavourites(Snippet, SnippetID, UserID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Post removed succesfuly!")
	http.Redirect(w, r, "/account/view", http.StatusSeeOther)

}
func (app *application) CommentaryPost(w http.ResponseWriter, r *http.Request) {
	var form commentaryForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form.CheckField(validator.NotBlank(form.Content), "content", "This field cannot be blank")
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "view.html", data)
		return
	}
	params := httprouter.ParamsFromContext(r.Context())
	SnippetIDStr := params.ByName("id")
	SnippetID, err := primitive.ObjectIDFromHex(SnippetIDStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	Author := map[string]string{
		app.sessionManager.GetString(r.Context(), "UserName"): app.sessionManager.GetString(r.Context(), "authenticatedUserID"),
	}

	form.Author = Author
	err = app.commentary.AddComentary(SnippetID, form.Author, form.Content)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Comment added succesfuly!")
	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%s", SnippetIDStr), http.StatusSeeOther)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, r, http.StatusOK, "signup.html", data)
}
func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Mathches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "signup.html", data)
		return
	}
	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "signup.html", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Your signup was successful. Please log in.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}
func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.html", data)
}
func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Mathches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.html", data)
		return
	}
	ObjectID, name, err := app.users.Authenticate(form.Email, form.Password)
	id := ObjectID.Hex()
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.html", data)
		} else {
			app.serverError(w, r, err)
		}
		return

	}
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	app.sessionManager.Put(r.Context(), "UserName", name)
	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)

}
func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
	}
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	idStr := app.sessionManager.GetString(r.Context(), "authenticatedUserID")
	if idStr == "" {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	user, err := app.users.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user
	app.render(w, r, http.StatusOK, "account.html", data)
}
func (app *application) otherAccountView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	idStr := params.ByName("id")
	if idStr == "" {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}
	if idStr == app.sessionManager.Get(r.Context(), "authenticatedUserID") {
		http.Redirect(w, r, "/account/view", http.StatusSeeOther)
		return
	}
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	user, err := app.users.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	data := app.newTemplateData(r)
	data.User = user
	app.render(w, r, http.StatusOK, "otherAccount.html", data)
}

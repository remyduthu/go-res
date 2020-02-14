package main

import (
	"strings"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store"
)

// BookHandler is a handler for book requests.
type BookHandler struct {
	BookStore *BookStore
	pattern   res.Pattern
}

// SetOption sets the res.Handler options.
func (h *BookHandler) SetOption(rh *res.Handler) {
	rh.Option(
		// Handler handels models
		res.Model,
		// Store handler that handles get requests and change events.
		store.Handler{Store: h.BookStore, Transformer: h},
		// Set call method handler, for updating the book's fields.
		res.Call("set", h.set),
		// On being registered to the res.Service, get the pattern (eg.
		// "library.book.$id") for this resource. This will be used in the
		// IDToRID transform function, to tell what external resource is
		// affected when a book is changed in the store.
		res.OnRegister(func(_ *res.Service, pattern string, _ res.Handler) {
			h.pattern = res.Pattern(pattern)
		}),
	)
}

// RIDToID transforms an external resource ID to a book ID used by the store.
func (h *BookHandler) RIDToID(rid string, pathParams map[string]string) string {
	return pathParams["id"]
}

// IDToRID transforms a book ID used by the store to an external resource ID.
func (h *BookHandler) IDToRID(id string, v interface{}) string {
	return string(h.pattern.ReplaceTag("id", id))
}

// Transform allows us to transform the stored book model before sending it off
// to external clients. In this example, we do no transformation.
func (h *BookHandler) Transform(id string, v interface{}) (interface{}, error) {
	// // We could convert the book to a type with a different JSON marshaler,
	// // or perhaps return a res.ErrNotFound if a deleted flag is set.
	// return BookWithDifferentJSONMarshaler(v.(Book)), nil
	return v, nil
}

// set handles set call requests on a book.
func (h *BookHandler) set(r res.CallRequest) {
	// Create a store write transaction.
	txn := h.BookStore.Write(r.PathParam("id"))
	defer txn.Close()

	// Get book value from store
	v, err := txn.Value()
	if err != nil {
		r.Error(err)
		return
	}
	book := v.(Book)

	// Unmarshal parameters to an anonymous struct
	var p struct {
		Title  *string `json:"title,omitempty"`
		Author *string `json:"author,omitempty"`
	}
	r.ParseParams(&p)

	// Validate title param
	if p.Title != nil {
		*p.Title = strings.TrimSpace(*p.Title)
		if *p.Title == "" {
			r.InvalidParams("Title must not be empty")
			return
		}
		book.Title = *p.Title
	}

	// Validate author param
	if p.Author != nil {
		*p.Author = strings.TrimSpace(*p.Author)
		if *p.Author == "" {
			r.InvalidParams("Author must not be empty")
			return
		}
		book.Author = *p.Author
	}

	// Update book in store.
	// This will produce a change event, if any fields were updated.
	// It might also produce events for the books collection, if the change
	// affects the sort order.
	err = txn.Update(book)
	if err != nil {
		r.Error(err)
		return
	}

	// Send success response
	r.OK(nil)
}

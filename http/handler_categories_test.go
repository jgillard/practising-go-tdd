package httptransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internal "github.com/jgillard/practising-go-tdd/internal"
)

func TestListCategories(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "abcdef", Name: "hostel", ParentID: "1234"},
			internal.Category{ID: "ghijkm", Name: "apartment", ParentID: "1234"},
		},
	}
	store := internal.NewInMemoryCategoryStore(&categoryList)
	server := NewServer(store, nil)

	t.Run("it returns a json category list", func(t *testing.T) {
		req := newGetRequest(t, "/categories")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.CategoryList
		unmarshallInterfaceFromBody(t, body, &got)

		want := categoryList
		assertDeepEqual(t, got, want)
		assertStringsEqual(t, got.Categories[0].ID, categoryList.Categories[0].ID)
		assertStringsEqual(t, got.Categories[0].Name, categoryList.Categories[0].Name)
		assertStringsEqual(t, got.Categories[0].ParentID, categoryList.Categories[0].ParentID)
		assertStringsEqual(t, got.Categories[1].ID, categoryList.Categories[1].ID)
		assertStringsEqual(t, got.Categories[1].Name, categoryList.Categories[1].Name)
		assertStringsEqual(t, got.Categories[1].ParentID, categoryList.Categories[1].ParentID)
	})
}

func TestGetCategory(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "1234", Name: "accommodation", ParentID: ""},
			internal.Category{ID: "2345", Name: "food and drink", ParentID: ""},
			internal.Category{ID: "abcdef", Name: "hostel", ParentID: "1234"},
			internal.Category{ID: "ghijkm", Name: "apartment", ParentID: "1234"},
		},
	}
	store := internal.NewInMemoryCategoryStore(&categoryList)
	server := NewServer(store, nil)

	t.Run("not-found failure reponse", func(t *testing.T) {
		req := newGetRequest(t, "/categories/5678")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusNotFound)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)
		assertBodyErrorTitle(t, body, internal.ErrorCategoryNotFound)
	})

	t.Run("get category with children", func(t *testing.T) {
		req := newGetRequest(t, "/categories/1234")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got CategoryGetResponse
		unmarshallInterfaceFromBody(t, body, &got)
		assertStringsEqual(t, got.ID, categoryList.Categories[0].ID)
		assertStringsEqual(t, got.Name, categoryList.Categories[0].Name)
		assertStringsEqual(t, got.ParentID, categoryList.Categories[0].ParentID)

		accomodationChildren := []internal.Category{
			categoryList.Categories[2],
			categoryList.Categories[3],
		}
		assertDeepEqual(t, got.Children, accomodationChildren)
	})

	t.Run("get category without children", func(t *testing.T) {
		req := newGetRequest(t, "/categories/2345")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got CategoryGetResponse
		unmarshallInterfaceFromBody(t, body, &got)
		assertStringsEqual(t, got.ID, categoryList.Categories[1].ID)
		assertStringsEqual(t, got.Name, categoryList.Categories[1].Name)
		assertStringsEqual(t, got.ParentID, categoryList.Categories[1].ParentID)
		assertNumbersEqual(t, len(got.Children), 0)
	})
}

func TestAddCategory(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "1234", Name: "existing category name", ParentID: ""},
			internal.Category{ID: "2345", Name: "existing subcategory name", ParentID: "1234"},
		},
	}
	store := internal.NewInMemoryCategoryStore(&categoryList)
	server := NewServer(store, nil)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			input      string
			want       int
			errorTitle string
		}{
			"invalid json": {
				input:      `"foo"`,
				want:       http.StatusBadRequest,
				errorTitle: errorInvalidJSON,
			},
			"name missing": {
				input:      `{}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorFieldMissing,
			},
			"duplicate name": {
				input:      `{"name":"existing category name"}`,
				want:       http.StatusConflict,
				errorTitle: internal.ErrorDuplicateCategoryName,
			},
			"invalid name": {
				input:      `{"name":"abc123!@£"}`,
				want:       http.StatusUnprocessableEntity,
				errorTitle: internal.ErrorInvalidCategoryName,
			},
			"parentID missing": {
				input:      `{"name":"valid name"}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorFieldMissing,
			},
			"parentID doesn't exist": {
				input:      `{"name":"foo", "parentID":"5678"}`,
				want:       http.StatusUnprocessableEntity,
				errorTitle: internal.ErrorParentIDNotFound,
			},
			"category would be >2 levels deep": {
				input:      `{"name":"foo", "parentID":"2345"}`,
				want:       http.StatusUnprocessableEntity,
				errorTitle: internal.ErrorCategoryTooNested,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				requestBody := strings.NewReader(c.input)
				req := newPostRequest(t, "/categories", requestBody)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := store.ListCategories()
				want := categoryList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("test success response & effect without parentID", func(t *testing.T) {
		categoryName := "new category name"
		parentID := ""

		cpr := CategoryPostRequest{
			Name:     categoryName,
			ParentID: &parentID,
		}
		payload, _ := json.Marshal(cpr)
		req := newPostRequest(t, "/categories", bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Category
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Name, categoryName)
		assertStringsEqual(t, got.ParentID, parentID)

		// check the store has been modified
		got = store.ListCategories().Categories[2]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Name, categoryName)
		assertStringsEqual(t, got.ParentID, parentID)

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s", got.ID))
	})

	t.Run("test success response & effect with parentID", func(t *testing.T) {
		categoryName := "another new category name"
		parentID := "1234"

		cpr := CategoryPostRequest{
			Name:     categoryName,
			ParentID: &parentID,
		}
		payload, _ := json.Marshal(cpr)
		req := newPostRequest(t, "/categories", bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Category
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Name, categoryName)
		assertStringsEqual(t, got.ParentID, parentID)

		// check the store has been modified
		got = store.ListCategories().Categories[3]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Name, categoryName)
		assertStringsEqual(t, got.ParentID, parentID)

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s", got.ID))
	})
}

func TestRenameCategory(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "1234", Name: "accommodation", ParentID: ""},
		},
	}
	store := internal.NewInMemoryCategoryStore(&categoryList)
	server := NewServer(store, nil)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			ID         string
			body       string
			want       int
			errorTitle string
		}{
			"invalid json": {
				ID:         "1234",
				body:       `{"foo":`,
				want:       http.StatusBadRequest,
				errorTitle: errorInvalidJSON,
			},
			"name missing": {
				ID:         "1234",
				body:       `{"foo":"bar"}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorFieldMissing,
			},
			"invalid name": {
				ID:         "1234",
				body:       `{"name":"foo/*!bar"}`,
				want:       http.StatusUnprocessableEntity,
				errorTitle: internal.ErrorInvalidCategoryName,
			},
			"duplicate name": {
				ID:         "1234",
				body:       `{"name":"accommodation"}`,
				want:       http.StatusConflict,
				errorTitle: internal.ErrorDuplicateCategoryName,
			},
			"ID not found": {
				ID:         "5678",
				body:       `{"name":"irrelevant"}`,
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorCategoryNotFound,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				requestBody := strings.NewReader(c.body)
				req := newPatchRequest(t, fmt.Sprintf("/categories/%s", c.ID), requestBody)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := store.ListCategories()
				want := categoryList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("test success responses & effect", func(t *testing.T) {
		newCatName := "new category name"
		requestBody, _ := json.Marshal(jsonName{Name: newCatName})
		req := newPatchRequest(t, "/categories/1234", bytes.NewReader(requestBody))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var responseBody internal.Category
		unmarshallInterfaceFromBody(t, body, &responseBody)

		renamedCategory := internal.Category{ID: "1234", Name: newCatName, ParentID: ""}
		assertStringsEqual(t, responseBody.ID, renamedCategory.ID)
		assertStringsEqual(t, responseBody.Name, renamedCategory.Name)
		assertStringsEqual(t, responseBody.ParentID, renamedCategory.ParentID)

		// check the store is updated
		got := store.ListCategories().Categories[0].Name
		want := renamedCategory.Name
		assertStringsEqual(t, got, want)
	})
}

func TestRemoveCategory(t *testing.T) {

	existingCategory := internal.Category{ID: "1234", Name: "accommodation"}
	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			existingCategory,
		},
	}
	store := internal.NewInMemoryCategoryStore(&categoryList)
	server := NewServer(store, nil)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			input      string
			want       int
			errorTitle string
		}{
			"category not found": {
				input:      "5678",
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorCategoryNotFound,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				req := newDeleteRequest(t, fmt.Sprintf("/categories/%s", c.input))
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := store.ListCategories()
				want := categoryList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("test success response & effect", func(t *testing.T) {
		req := newDeleteRequest(t, "/categories/1234")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)
		assertBodyJSONIsStatus(t, body, statusDeleted)

		// check store is updated
		got := len(store.ListCategories().Categories)
		want := 0
		assertNumbersEqual(t, got, want)
	})
}

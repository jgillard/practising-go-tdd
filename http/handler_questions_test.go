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

func TestListQuestionsForCategory(t *testing.T) {

	questionList := internal.QuestionList{
		Questions: []internal.Question{
			internal.Question{ID: "1", Title: "how many nights?", CategoryID: "1234", Type: "number"},
			internal.Question{ID: "2", Title: "which meal?", CategoryID: "5678", Type: "string", Options: internal.OptionList{
				{ID: "1", Title: "brekkie"},
			}},
		},
	}
	questionStore := internal.NewInMemoryQuestionStore(&questionList)
	server := NewServer(nil, questionStore)

	t.Run("it returns a json question list for a category", func(t *testing.T) {
		req := newGetRequest(t, "/categories/1234/questions")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.QuestionList
		unmarshallInterfaceFromBody(t, body, &got)
		want := questionList.Questions[0]
		assertDeepEqual(t, got.Questions[0], want)
	})

	t.Run("it returns a json question list for a category with options", func(t *testing.T) {
		req := newGetRequest(t, "/categories/5678/questions")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.QuestionList
		unmarshallInterfaceFromBody(t, body, &got)
		want := questionList.Questions[1]
		assertDeepEqual(t, got.Questions[0], want)
	})

	t.Run("it returns an empty json question list when no questions exist for category", func(t *testing.T) {
		req := newGetRequest(t, "/categories/2345/questions")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.QuestionList
		unmarshallInterfaceFromBody(t, body, &got)
		want := len(got.Questions)
		assertNumbersEqual(t, want, 0)
	})
}

func TestGetQuestion(t *testing.T) {

	questionList := internal.QuestionList{
		Questions: []internal.Question{
			internal.Question{ID: "2", Title: "which meal?", CategoryID: "5678", Type: "string", Options: internal.OptionList{
				{ID: "1", Title: "brekkie"},
			}},
		},
	}
	questionStore := internal.NewInMemoryQuestionStore(&questionList)
	server := NewServer(nil, questionStore)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			path       string
			want       int
			errorTitle string
		}{
			"ID not found": {
				path:       "/categories/5678/questions/1",
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorQuestionNotFound,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				req := newGetRequest(t, c.path)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)
				assertBodyErrorTitle(t, body, c.errorTitle)
			})
		}
	})

	t.Run("get category with children", func(t *testing.T) {
		req := newGetRequest(t, "/categories/5678/questions/2")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Question
		unmarshallInterfaceFromBody(t, body, &got)
		assertStringsEqual(t, got.ID, questionList.Questions[0].ID)
		assertStringsEqual(t, got.Title, questionList.Questions[0].Title)
		assertStringsEqual(t, got.CategoryID, questionList.Questions[0].CategoryID)
		assertStringsEqual(t, got.Type, questionList.Questions[0].Type)
		assertDeepEqual(t, got.Options, questionList.Questions[0].Options)
	})
}

func TestAddQuestion(t *testing.T) {

	t.Run("test failure responses & effect", func(t *testing.T) {
		categoryList := internal.CategoryList{
			Categories: []internal.Category{
				internal.Category{ID: "1234", Name: "foo", ParentID: ""},
			},
		}
		questionList := internal.QuestionList{
			Questions: []internal.Question{
				internal.Question{ID: "1", Title: "how many nights?", CategoryID: "1234", Type: "number"},
				internal.Question{ID: "2", Title: "which meal?", CategoryID: "5678", Type: "string", Options: internal.OptionList{
					{ID: "1", Title: "brekkie"},
				}},
			},
		}
		categoryStore := internal.NewInMemoryCategoryStore(&categoryList)
		questionStore := internal.NewInMemoryQuestionStore(&questionList)
		server := NewServer(categoryStore, questionStore)

		cases := map[string]struct {
			path       string
			input      string
			want       int
			errorTitle string
		}{
			"invalid json": {
				path:       "/categories/1234/questions",
				input:      `{"foo":`,
				want:       http.StatusBadRequest,
				errorTitle: errorInvalidJSON,
			},
			"title is empty": {
				path:       "/categories/1234/questions",
				input:      `{"title":"", "type":"number"}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorTitleEmpty,
			},
			"title is duplicate": {
				path:       "/categories/1234/questions",
				input:      `{"title":"how many nights?", "type":"number"}`,
				want:       http.StatusConflict,
				errorTitle: internal.ErrorDuplicateTitle,
			},
			"type is empty": {
				path:       "/categories/1234/questions",
				input:      `{"title":"foo", "type":""}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorTypeEmpty,
			},
			"type doesn't exist": {
				path:       "/categories/1234/questions",
				input:      `{"title":"foo", "type":"foo"}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorInvalidType,
			},
			"options is not list type": {
				path:       "/categories/1234/questions",
				input:      `{"title":"foo", "type":"string", "options":""}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorOptionsInvalid,
			},
			"options has duplicate": {
				path:       "/categories/1234/questions",
				input:      `{"title":"foo", "type":"string", "options":["foo", "foo"]}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorDuplicateOption,
			},
			"options contains empty string": {
				path:       "/categories/1234/questions",
				input:      `{"title":"foo", "type":"string", "options":[""]}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorOptionEmpty,
			},
			"category doesn't exist": {
				path:       "/categories/5678/questions",
				input:      `{"title":"foo", "type":"string"}`,
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorCategoryNotFound,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				requestBody := strings.NewReader(c.input)
				req := newPostRequest(t, c.path, requestBody)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := questionStore.ListQuestions()
				want := questionList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("add type:number question", func(t *testing.T) {
		questionStore := internal.NewInMemoryQuestionStore(nil)
		server := NewServer(nil, questionStore)

		categoryID := "1"
		title := "how many nights?"
		optionType := "number"

		qpr := internal.QuestionPostRequest{
			Title:   title,
			Type:    optionType,
			Options: nil,
		}
		payload, _ := json.Marshal(qpr)
		path := fmt.Sprintf("/categories/%s/questions", categoryID)
		req := newPostRequest(t, path, bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Question
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		// Options should not be present in response
		assertOptionsNil(t, got.Options)

		// check the store has been modified
		got = questionStore.ListQuestions().Questions[0]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s/questions/%s", categoryID, got.ID))
	})

	t.Run("add type:string question without options", func(t *testing.T) {
		questionStore := internal.NewInMemoryQuestionStore(nil)
		server := NewServer(nil, questionStore)

		categoryID := "1"
		title := "what number?"
		optionType := "number"

		qpr := internal.QuestionPostRequest{
			Title:   title,
			Type:    optionType,
			Options: nil,
		}
		payload, _ := json.Marshal(qpr)
		path := fmt.Sprintf("/categories/%s/questions", categoryID)
		req := newPostRequest(t, path, bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Question
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		assertOptionsNil(t, got.Options)

		// check the store has been modified
		got = questionStore.ListQuestions().Questions[0]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		assertOptionsNil(t, got.Options)

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s/questions/%s", categoryID, got.ID))
	})

	t.Run("add type:string question with empty options", func(t *testing.T) {
		questionStore := internal.NewInMemoryQuestionStore(nil)
		server := NewServer(nil, questionStore)

		categoryID := "1"
		title := "which meal?"
		optionType := "string"
		options := []string{}

		qpr := internal.QuestionPostRequest{
			Title:   title,
			Type:    optionType,
			Options: &options,
		}
		payload, _ := json.Marshal(qpr)
		path := fmt.Sprintf("/categories/%s/questions", categoryID)
		req := newPostRequest(t, path, bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Question
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		// options should be an empty list
		assertDeepEqual(t, got.Options, internal.OptionList{})

		// check the store has been modified
		got = questionStore.ListQuestions().Questions[0]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		assertDeepEqual(t, got.Options, internal.OptionList{})

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s/questions/%s", categoryID, got.ID))
	})

	t.Run("add type:string question with options", func(t *testing.T) {
		questionStore := internal.NewInMemoryQuestionStore(nil)
		server := NewServer(nil, questionStore)

		categoryID := "1"
		title := "which meal?"
		optionType := "string"
		options := []string{"brekkie", "lunch"}

		qpr := internal.QuestionPostRequest{
			Title:   title,
			Type:    optionType,
			Options: &options,
		}
		payload, _ := json.Marshal(qpr)
		path := fmt.Sprintf("/categories/%s/questions", categoryID)
		req := newPostRequest(t, path, bytes.NewReader(payload))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		assertStatusCode(t, result.StatusCode, http.StatusCreated)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var got internal.Question
		unmarshallInterfaceFromBody(t, body, &got)

		// check the response
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		assertNumbersEqual(t, len(got.Options), 2)
		assertIsXid(t, got.Options[0].ID)
		assertIsXid(t, got.Options[1].ID)
		assertStringsEqual(t, got.Options[0].Title, options[0])
		assertStringsEqual(t, got.Options[1].Title, options[1])

		// check the store has been modified
		got = questionStore.ListQuestions().Questions[0]
		assertIsXid(t, got.ID)
		assertStringsEqual(t, got.Title, title)
		assertStringsEqual(t, got.CategoryID, categoryID)
		assertStringsEqual(t, got.Type, optionType)
		assertNumbersEqual(t, len(got.Options), 2)
		assertIsXid(t, got.Options[0].ID)
		assertIsXid(t, got.Options[1].ID)
		assertStringsEqual(t, got.Options[0].Title, options[0])
		assertStringsEqual(t, got.Options[1].Title, options[1])

		// get ID from store and check that's in returned Location header
		assertStringsEqual(t, result.Header.Get("Location"), fmt.Sprintf("/categories/%s/questions/%s", categoryID, got.ID))
	})
}

func TestRenameQuestion(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "1234", Name: "foo", ParentID: ""},
			internal.Category{ID: "2345", Name: "bar", ParentID: ""},
		},
	}
	questionList := internal.QuestionList{
		Questions: []internal.Question{
			internal.Question{ID: "1", Title: "how many nuggets?", CategoryID: "1234", Type: "number"},
			internal.Question{ID: "2", Title: "how much nougat?", CategoryID: "1234", Type: "number"},
			internal.Question{ID: "3", Title: "how much nougat?", CategoryID: "2345", Type: "number"},
		},
	}
	categoryStore := internal.NewInMemoryCategoryStore(&categoryList)
	questionStore := internal.NewInMemoryQuestionStore(&questionList)
	server := NewServer(categoryStore, questionStore)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			path       string
			input      string
			want       int
			errorTitle string
		}{
			"invalid json": {
				path:       "/categories/1234/questions/1",
				input:      `{"foo":}`,
				want:       http.StatusBadRequest,
				errorTitle: errorInvalidJSON,
			},
			"missing title": {
				path:       "/categories/1234/questions/1",
				input:      `{}`,
				want:       http.StatusBadRequest,
				errorTitle: internal.ErrorFieldMissing,
			},
			"invalid title": {
				path:       "/categories/1234/questions/1",
				input:      `{"title":"foo/*!bar"}`,
				want:       http.StatusUnprocessableEntity,
				errorTitle: internal.ErrorInvalidTitle,
			},
			"duplicate title": {
				path:       "/categories/1234/questions/1",
				input:      `{"title":"how much nougat?"}`,
				want:       http.StatusConflict,
				errorTitle: internal.ErrorDuplicateTitle,
			},
			"category doesn't exist": {
				path:       "/categories/5678/questions/1",
				input:      `{"title":"irrelevant"}`,
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorCategoryNotFound,
			},
			"ID not found": {
				path:       "/categories/1234/questions/4",
				input:      `{"title":"irrelevant"}`,
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorQuestionNotFound,
			},
			"question does not belong to category": {
				path:       "/categories/1234/questions/3",
				input:      `{"title":"irrelevant"}`,
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorQuestionDoesntBelongToCategory,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				requestBody := strings.NewReader(c.input)
				req := newPatchRequest(t, c.path, requestBody)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)
				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := questionStore.ListQuestions()
				want := questionList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("test success responses & effect", func(t *testing.T) {
		newQuestionTitle := "whattup world?"
		requestBody, err := json.Marshal(jsonTitle{Title: newQuestionTitle})
		if err != nil {
			t.Fatal(err)
		}
		req := newPatchRequest(t, "/categories/1234/questions/1", bytes.NewReader(requestBody))
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check the response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

		var responseBody internal.Question
		unmarshallInterfaceFromBody(t, body, &responseBody)

		renamedQuestion := internal.Question{ID: "1", Title: newQuestionTitle, CategoryID: "1234", Type: "number"}
		assertStringsEqual(t, responseBody.ID, renamedQuestion.ID)
		assertStringsEqual(t, responseBody.Title, renamedQuestion.Title)
		assertStringsEqual(t, responseBody.CategoryID, renamedQuestion.CategoryID)
		assertStringsEqual(t, responseBody.Type, renamedQuestion.Type)

		// check the store is updated
		got := questionStore.ListQuestions().Questions[0].Title
		want := renamedQuestion.Title
		assertStringsEqual(t, got, want)
	})
}

func TestRemoveQuestion(t *testing.T) {

	categoryList := internal.CategoryList{
		Categories: []internal.Category{
			internal.Category{ID: "1234", Name: "foo", ParentID: ""},
			internal.Category{ID: "2345", Name: "bar", ParentID: ""},
		},
	}
	questionList := internal.QuestionList{
		Questions: []internal.Question{
			internal.Question{ID: "1", Title: "how many nuggets?", CategoryID: "1234", Type: "number"},
		},
	}
	categoryStore := internal.NewInMemoryCategoryStore(&categoryList)
	questionStore := internal.NewInMemoryQuestionStore(&questionList)
	server := NewServer(categoryStore, questionStore)

	t.Run("test failure responses & effect", func(t *testing.T) {
		cases := map[string]struct {
			path       string
			want       int
			errorTitle string
		}{
			"category doesn't exist": {
				path:       "/categories/5678/questions/2",
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorCategoryNotFound,
			},
			"question doesn't exist": {
				path:       "/categories/1234/questions/2",
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorQuestionNotFound,
			},
			"question doesn't belong to category": {
				path:       "/categories/2345/questions/1",
				want:       http.StatusNotFound,
				errorTitle: internal.ErrorQuestionDoesntBelongToCategory,
			},
		}

		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				req := newDeleteRequest(t, c.path)
				res := httptest.NewRecorder()

				server.ServeHTTP(res, req)
				result := res.Result()
				body := readBodyJSON(t, result.Body)

				// check the response
				assertStatusCode(t, result.StatusCode, c.want)
				assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)

				assertBodyErrorTitle(t, body, c.errorTitle)

				// check the store is unmodified
				got := questionStore.ListQuestions()
				want := questionList
				assertDeepEqual(t, got, want)
			})
		}
	})

	t.Run("test success response & effect", func(t *testing.T) {
		req := newDeleteRequest(t, "/categories/1234/questions/1")
		res := httptest.NewRecorder()

		server.ServeHTTP(res, req)
		result := res.Result()
		body := readBodyJSON(t, result.Body)

		// check response
		assertStatusCode(t, result.StatusCode, http.StatusOK)
		assertContentType(t, result.Header.Get(contentTypeKey), jsonContentType)
		assertBodyJSONIsStatus(t, body, statusDeleted)

		// check store is updated
		got := len(questionStore.ListQuestions().Questions)
		want := 0
		assertNumbersEqual(t, got, want)
	})
}

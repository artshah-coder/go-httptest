package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	FILENAME = "dataset.xml"
)

func SearchServer(resp http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != "123" {
		resp.WriteHeader(http.StatusUnauthorized)
		result, err := json.Marshal(SearchErrorResponse{Error: "Unauthorized"})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	url, err := url.Parse(r.RequestURI)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{
			Error: fmt.Sprintf("Bad URI request %v", r.RequestURI),
		})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	req := SearchRequest{}

	req.Limit, err = strconv.Atoi(url.Query().Get("limit"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{
			Error: "Bad limit param value: must be an integer!",
		})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	req.Offset, err = strconv.Atoi(url.Query().Get("offset"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{
			Error: "Bad offset param value: must be an integer!",
		})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	if req.Offset >= req.Limit {
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{
			Error: "Bad offset param value: must be less than limit param!",
		})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	req.OrderBy, err = strconv.Atoi(url.Query().Get("order_by"))
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{
			Error: "Bad order_by param value: must be an integer!",
		})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	req.OrderField = url.Query().Get("order_field")
	req.Query = url.Query().Get("query")

	result := make([]User, 0, req.Limit)

	var tmp struct {
		Id         int    `xml:"id"`
		First_name string `xml:"first_name"`
		Last_name  string `xml:"last_name"`
		Age        int    `xml:"age"`
		About      string `xml:"about"`
		Gender     string `xml:"gender"`
	}

	dataset, err := os.Open(FILENAME)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		result, err := json.Marshal(SearchErrorResponse{
			Error: "Internal server error",
		})
		if err != nil {
			fmt.Println(err)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}
	defer dataset.Close()

	decoder := xml.NewDecoder(dataset)
	i := 0
PARSE:
	for i < req.Limit {
		t, err := decoder.Token()
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			result, err := json.Marshal(SearchErrorResponse{
				Error: "Internal server error",
			})
			if err != nil {
				fmt.Println(err)
			}
			fmt.Fprintf(resp, "%s", string(result))
			return
		}

		switch et := t.(type) {
		case xml.StartElement:
			if et.Name.Local == "row" {
				if err := decoder.DecodeElement(&tmp, &et); err != nil {
					resp.WriteHeader(http.StatusInternalServerError)
					result, err := json.Marshal(SearchErrorResponse{
						Error: "Internal server error",
					})
					if err != nil {
						fmt.Println(err)
					}
					fmt.Fprintf(resp, "%s", string(result))
					return
				}
				if strings.Contains(tmp.First_name, req.Query) ||
					strings.Contains(tmp.Last_name, req.Query) ||
					strings.Contains(tmp.About, req.Query) {

					result = append(result, User{
						Id:     tmp.Id,
						Name:   tmp.First_name + " " + tmp.Last_name,
						Age:    tmp.Age,
						About:  tmp.About,
						Gender: tmp.Gender,
					})
					i++
				}
			}
		case xml.EndElement:
			if et.Name.Local == "root" {
				break PARSE
			}
		}
	}

	switch req.OrderField {
	case "Id":
		switch req.OrderBy {
		case OrderByAsc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Id < result[j].Id
			})
		case OrderByDesc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Id > result[j].Id
			})
		case OrderByAsIs:
		default:
		}
	case "Age":
		switch req.OrderBy {
		case OrderByAsc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Age < result[j].Age
			})
		case OrderByDesc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Age > result[j].Age
			})
		case OrderByAsIs:
		default:
		}
	case "Name":
		switch req.OrderBy {
		case OrderByAsc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Name < result[j].Name
			})
		case OrderByDesc:
			sort.Slice(result, func(i, j int) bool {
				return result[i].Name > result[j].Name
			})
		case OrderByAsIs:
		default:
		}
	default:
		resp.WriteHeader(http.StatusBadRequest)
		result, err := json.Marshal(SearchErrorResponse{Error: "ErrorBadOrderField"})
		if err != nil {
			fmt.Println(err)
			resp.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Fprintf(resp, "%s", string(result))
		return
	}

	if req.Offset >= len(result) {
		result = make([]User, 0)
	} else {
		result = result[req.Offset:]
	}

	resp.WriteHeader(http.StatusOK)
	respData, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
		resp.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintf(resp, "%s", string(respData))
}

func SearchServerErrors(resp http.ResponseWriter, r *http.Request) {
	url, _ := url.Parse(r.RequestURI)
	req := SearchRequest{}

	req.Offset, _ = strconv.Atoi(url.Query().Get("offset"))
	switch req.Offset {
	case 5:
		resp.WriteHeader(http.StatusOK)
	case 10:
		resp.WriteHeader(http.StatusBadRequest)
	case 15:
		resp.WriteHeader(http.StatusInternalServerError)
		return
	case 20:
		time.Sleep(20 * time.Microsecond)
		resp.WriteHeader(http.StatusOK)
	}
	fmt.Fprintf(resp, `{"status": 400`)
}

type TestCase struct {
	Request  SearchRequest
	Response *SearchResponse
	Error    error
}

func TestFindUsers(t *testing.T) {
	cases := []TestCase{
		// Unauthorized test case
		TestCase{
			Request:  SearchRequest{},
			Response: nil,
			Error:    errors.New("Bad AccessToken"),
		},
		TestCase{
			Request: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "An",
				OrderField: "Id",
				OrderBy:    0,
			},
			Response: &SearchResponse{
				Users: []User{
					User{
						Id:   16,
						Name: "Annie Osborn",
						Age:  35,
						About: "Consequat fugiat veniam commodo nisi nostrud culpa pariatur. " +
							"Aliquip velit adipisicing dolor et nostrud. " +
							"Eu nostrud officia velit eiusmod ullamco duis eiusmod ad non do quis.\n",
						Gender: "female",
					},
				},
				NextPage: true,
			},
			Error: nil,
		},
		TestCase{
			Request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "Annet",
				OrderField: "Id",
				OrderBy:    0,
			},
			Response: &SearchResponse{
				Users:    []User{},
				NextPage: false,
			},
			Error: nil,
		},
		TestCase{
			Request: SearchRequest{
				OrderField: "About",
			},
			Response: nil,
			Error:    errors.New("OrderFeld About invalid"),
		},
		TestCase{
			Request: SearchRequest{
				Offset: -1,
			},
			Response: nil,
			Error:    errors.New("offset must be > 0"),
		},
		TestCase{
			Request: SearchRequest{
				Limit: -1,
			},
			Response: nil,
			Error:    errors.New("limit must be > 0"),
		},
		TestCase{
			Request: SearchRequest{
				Limit:      19,
				Offset:     20,
				OrderField: "Id",
			},
			Response: nil,
			Error:    errors.New("unknown bad request error: Bad offset param value: must be less than limit param!"),
		},
		TestCase{
			Request: SearchRequest{
				Limit:  30,
				Offset: 30,
			},
			Response: nil,
			Error:    errors.New("unknown bad request error: Bad offset param value: must be less than limit param!"),
		},
		// Unknown client error test case
		TestCase{
			Request:  SearchRequest{},
			Response: nil,
			Error:    errors.New(`unknown error Get "?limit=1&offset=0&order_by=0&order_field=&query=": unsupported protocol scheme ""`),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srv := &SearchClient{
		URL: ts.URL,
	}

	for caseNum, item := range cases {
		if caseNum == 0 {
			// Unauthorized test case
			searchResponse, err := srv.FindUsers(SearchRequest{})
			if !reflect.DeepEqual(searchResponse, item.Response) || err.Error() != item.Error.Error() {
				t.Errorf("[%d case] got: (%v %v), expected: (%v %v)\n",
					caseNum, searchResponse, err, item.Response, item.Error)
			}
			srv.AccessToken = "123"
			continue
		}
		if caseNum == len(cases)-1 {
			// Unknown client error test case
			srv.URL = ""
			searchResponse, err := srv.FindUsers(SearchRequest{})
			if !reflect.DeepEqual(searchResponse, item.Response) || err.Error() != item.Error.Error() {
				t.Errorf("[%d case] got: (%v %v), expected: (%v %v)\n",
					caseNum, searchResponse, err, item.Response, item.Error)
			}
			break
		}
		searchResponse, err := srv.FindUsers(item.Request)
		if !reflect.DeepEqual(searchResponse, item.Response) {
			t.Errorf("[%d case] got: (%v %v), expected: (%v %v)\n",
				caseNum, searchResponse, err, item.Response, item.Error)
			continue
		}

		switch {
		case err == nil:
			if item.Error != nil {
				t.Errorf("[%d case] got: (%v %v), expected: (%v %v)\n",
					caseNum, searchResponse, err, item.Response, item.Error)
			}
		case err != nil:
			if item.Error == nil || err.Error() != item.Error.Error() {
				t.Errorf("[%d case] got: (%v %v), expected: (%v %v)\n",
					caseNum, searchResponse, err, item.Response, item.Error)
			}
		}
	}
	ts.Close()

	// Broken json test cases
	errorTS := httptest.NewServer(http.HandlerFunc(SearchServerErrors))
	srv.URL = errorTS.URL
	e := errors.New("cant unpack result json: unexpected end of JSON input")
	searchResponse, err := srv.FindUsers(SearchRequest{
		Offset: 5,
	})
	if searchResponse != nil || err.Error() != e.Error() {
		t.Errorf("[broken json 200 OK case] got: (%v %v), expected: (%v %v)\n",
			searchResponse, err, nil, e)
	}

	e = errors.New("cant unpack error json: unexpected end of JSON input")
	searchResponse, err = srv.FindUsers(SearchRequest{
		Offset: 10,
	})
	if searchResponse != nil || err.Error() != e.Error() {
		t.Errorf("[broken json 400 Bad Request case] got: (%v %v), expected: (%v %v)\n",
			searchResponse, err, nil, e)
	}

	// http.StatusInternalServerError test case
	e = errors.New("SearchServer fatal error")
	searchResponse, err = srv.FindUsers(SearchRequest{
		Offset: 15,
	})
	if searchResponse != nil || err.Error() != e.Error() {
		t.Errorf("[500 Internal Server Error case] got: (%v %v), expected: (%v %v)\n",
			searchResponse, err, nil, e)
	}

	// Timeout test case
	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Microsecond
	e = errors.New("timeout for limit=1&offset=20&order_by=0&order_field=&query=")
	searchResponse, err = srv.FindUsers(SearchRequest{
		Offset: 20,
	})
	if searchResponse != nil || err.Error() != e.Error() {
		t.Errorf("[timeout case] got: (%v %v), expected: (%v %v)\n",
			searchResponse, err, nil, e)
	}

	errorTS.Close()
}

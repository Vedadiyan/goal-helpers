package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/vedadiyan/goal/pkg/http"
	"github.com/vedadiyan/goal/pkg/protoutil"
	"github.com/vedadiyan/gql/pkg/sql"
	"google.golang.org/protobuf/proto"
)

type Result struct {
	result any
}

type Query struct {
	query string
}

func New(query string) *Query {
	q := Query{
		query: query,
	}
	return &q
}

func (q Query) exec(data map[string]any) (any, error) {
	if q.query == "" {
		return data, nil
	}
	query := sql.New(data)
	err := query.Prepare(q.query)
	if err != nil {
		return nil, err
	}
	r, err := query.Exec()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (q Query) OnProtobuf(m proto.Message) (*Result, error) {
	req, err := protoutil.Marshal(m)
	if err != nil {
		return nil, err
	}
	r, err := q.exec(req)
	if err != nil {
		return nil, err
	}
	res := Result{
		result: r,
	}
	return &res, nil
}

func (q Query) OnJSON(bytes []byte) (*Result, error) {
	var mapper map[string]any
	err := json.Unmarshal(bytes, &mapper)
	if err != nil {
		return nil, err
	}
	r, err := q.exec(mapper)
	if err != nil {
		return nil, err
	}
	res := Result{
		result: r,
	}
	return &res, nil
}

func (q Query) OnJSONStream(stream io.ReadCloser) (*Result, error) {
	bytes, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	return q.OnJSON(bytes)
}

func (r Result) ToJSON() (http.JSON, error) {
	json, err := json.Marshal(filter(r.result.(map[string]any)))
	if err != nil {
		return "", err
	}
	return http.JSON(json), nil
}

func (r Result) ToProtobuf(m proto.Message) error {
	err := protoutil.Unmarshal(r.result.(map[string]any), m)
	if err != nil {
		return err
	}
	return nil
}

func (r Result) QueryParams() (map[string][]string, error) {
	flattened := FlattenMap(r.result.(map[string]any))
	pattern := `\{.*?\}`
	re := regexp.MustCompile(pattern)
	out := make(map[string][]string)
	for key, value := range flattened {
		n := strings.Count(key, "$")
		if n == 0 {
			continue
		}
		if n > 1 {
			return nil, fmt.Errorf("ambiguous query string parameter definition")
		}
		k := strings.Split(key, "$")[1]
		n = strings.Count(k, ".")
		if n == 0 {
			out[k] = make([]string, 1)
			out[k][0] = fmt.Sprintf("%v", value)
			continue
		}
		dim := re.FindAll([]byte(k), -1)
		if len(dim) == 0 {
			return nil, fmt.Errorf("expected index")
		}
		if len(dim) > 1 {
			return nil, fmt.Errorf("multi-dimensional arrays are not exportable to query string parameters")
		}
		k = strings.Replace(k, string(dim[0]), "", 1)
		if _, ok := out[k]; !ok {
			out[k] = make([]string, 0)
		}
		out[k] = append(out[k], fmt.Sprintf("%v", value))
	}
	return out, nil
}

func (r Result) RouteValues() (map[string]string, error) {
	flattened := FlattenMap(r.result.(map[string]any))
	out := make(map[string]string)
	for key, value := range flattened {
		n := strings.Count(key, ":")
		if n == 0 {
			continue
		}
		if n > 1 {
			return nil, fmt.Errorf("ambiguous route value definition")
		}
		k := strings.Split(key, ":")[1]
		n = strings.Count(k, ".")
		if n == 0 {
			return nil, fmt.Errorf("ambiguous route value definition")
		}
		out[k] = fmt.Sprintf("%v", value)
	}
	return out, nil
}

func MakeReq[TReq proto.Message](m TReq, reqMapper []byte) (http.RouteValues, http.Query, http.JSON, error) {
	r, err := New(string(reqMapper)).OnProtobuf(m)
	if err != nil {
		return nil, nil, "", err
	}
	routeValue, err := r.RouteValues()
	if err != nil {
		return nil, nil, "", err
	}
	query, err := r.QueryParams()
	if err != nil {
		return nil, nil, "", err
	}
	json, err := r.ToJSON()
	if err != nil {
		return nil, nil, "", err
	}
	return routeValue, query, json, nil
}

func MakeRes[TRes proto.Message](data io.ReadCloser, resMapper []byte) (TRes, error) {
	r, err := New(string(resMapper)).OnJSONStream(data)
	if err != nil {
		return *new(TRes), err
	}
	var m TRes
	err = r.ToProtobuf(m)
	if err != nil {
		return *new(TRes), err
	}
	return m, nil
}

func filter(data map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range FlattenMap(data) {
		if strings.Contains(key, "$") || strings.Contains(key, ":") {
			continue
		}
		out[key] = value
	}
	return UnFlatten(out)
}

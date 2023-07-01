package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/vedadiyan/goal/pkg/http"
	"github.com/vedadiyan/goal/pkg/protoutil"
	"github.com/vedadiyan/gql/pkg/sql"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Result struct {
	result any
}

type Query struct {
	query string
}

type Request[T any] struct {
	RouteValues map[string]string
	QueryParams map[string][]string
	Body        T
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
	return q.OnJSON([]byte(string(bytes)))
}

func (r Result) ToJSON() (http.JSON, error) {
	var data map[string]any
	if value, ok := r.result.([]any); ok {
		if len(value) == 0 {
			return "", nil
		}
		value, ok := value[0].(map[string]any)
		if !ok {
			return "", fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	} else {
		value, ok := r.result.(map[string]any)
		if !ok {
			return "", fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	}
	res := filter(data)
	if res == nil {
		return http.JSON(""), nil
	}
	json, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return http.JSON(json), nil
}

func (r Result) ToURLEncoded() (http.URLEncoded, error) {
	var data map[string]any
	if value, ok := r.result.([]any); ok {
		if len(value) == 0 {
			return nil, nil
		}
		value, ok := value[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	} else {
		value, ok := r.result.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	}
	res := filter(data)
	if res == nil {
		return nil, nil
	}
	urlEncoded := url.Values{}
	for key, value := range res.(map[string]any) {
		urlEncoded.Add(key, fmt.Sprintf("%v", value))
	}
	return urlEncoded, nil
}

func (r Result) ToProtobuf(m proto.Message) error {
	// if len(r.result.([]any)) == 0 {
	// 	return nil
	// }
	err := protoutil.Unmarshal(r.result, m)
	if err != nil {
		return err
	}
	return nil
}

func (r Result) QueryParams() (map[string][]string, error) {
	var data map[string]any
	if value, ok := r.result.([]any); ok {
		if len(value) == 0 {
			return nil, nil
		}
		value, ok := value[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	} else {
		value, ok := r.result.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	}
	flattened := FlattenMap(data)
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
			if value != nil {
				out[k] = make([]string, 1)
				out[k][0] = fmt.Sprintf("%v", value)
			}
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
		if value != nil {
			out[k] = append(out[k], fmt.Sprintf("%v", value))
		}
	}
	return out, nil
}

func (r Result) RouteValues() (map[string]string, error) {
	var data map[string]any
	if value, ok := r.result.([]any); ok {
		if len(value) == 0 {
			return nil, nil
		}
		value, ok := value[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	} else {
		value, ok := r.result.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map but recieved %T", value)
		}
		data = value
	}
	flattened := FlattenMap(data)
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

func ToJSONReq[TReq proto.Message](m TReq, reqMapper []byte) (*Request[http.JSON], error) {
	r, err := New(string(reqMapper)).OnProtobuf(m)
	if err != nil {
		return nil, err
	}
	routeValue, err := r.RouteValues()
	if err != nil {
		return nil, err
	}
	query, err := r.QueryParams()
	if err != nil {
		return nil, err
	}
	json, err := r.ToJSON()
	if err != nil {
		return nil, err
	}
	req := Request[http.JSON]{
		RouteValues: routeValue,
		QueryParams: query,
		Body:        json,
	}
	return &req, nil
}

func ToURLEncodedReq[TReq proto.Message](m TReq, reqMapper []byte) (*Request[http.URLEncoded], error) {
	r, err := New(string(reqMapper)).OnProtobuf(m)
	if err != nil {
		return nil, err
	}
	routeValue, err := r.RouteValues()
	if err != nil {
		return nil, err
	}
	query, err := r.QueryParams()
	if err != nil {
		return nil, err
	}
	values, err := r.ToURLEncoded()
	if err != nil {
		return nil, err
	}

	req := Request[http.URLEncoded]{
		RouteValues: routeValue,
		QueryParams: query,
		Body:        values,
	}
	return &req, nil
}

func Exec[TReq proto.Message, TRes proto.Message](m TReq, reqMapper []byte, r TRes) error {
	res, err := New(string(reqMapper)).OnProtobuf(m)
	if err != nil {
		return err
	}
	err = res.ToProtobuf(r)
	if err != nil {
		return err
	}
	return nil
}

func FromJSONRes[TRes proto.Message](data io.ReadCloser, resMapper []byte, m TRes) error {
	r, err := New(string(resMapper)).OnJSONStream(data)
	if err != nil {
		return err
	}
	err = r.ToProtobuf(m)
	if err != nil {
		return err
	}
	return nil
}

func GetJSONReq[T proto.Message](c *fiber.Ctx, req T) error {
	values := make(map[string]any)
	for _, key := range c.Route().Params {
		values[key] = c.Params(key)
	}
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		values[string(key)] = string(value)
	})
	if len(c.Body()) != 0 {
		c.BodyParser(&values)
	}
	out, err := json.Marshal(values)
	if err != nil {
		return err
	}
	err = protojson.Unmarshal(out, req)
	if err != nil {
		return err
	}
	return nil
}

func SendJSONRes[T proto.Message](data T, c *fiber.Ctx) error {
	out, err := protojson.Marshal(data)
	if err != nil {
		return err
	}
	c.Response().Header.Add("content-type", "application/json")
	return c.Send(out)
}

func filter(data any) any {
	// out := make(map[string]any)
	// for key, value := range FlattenMap(data) {
	// 	if strings.Contains(key, "$") || strings.Contains(key, ":") {
	// 		continue
	// 	}
	// 	out[key] = value
	// }
	// return UnFlatten(out)
	switch t := data.(type) {
	case map[string]any:
		{
			if len(t) == 0 {
				return t
			}
			copy := make(map[string]any)
			for key, value := range t {
				if strings.Contains(key, "$") || strings.Contains(key, ":") {
					continue
				}
				res := filter(value)
				copy[key] = res
			}
			if len(copy) == 0 {
				return nil
			}
			return copy
		}
	case []any:
		{
			if len(t) == 0 {
				return t
			}
			copy := make([]any, 0)
			for _, item := range t {
				res := filter(item)
				if res != nil {
					copy = append(copy, res)
				}
			}
			if len(copy) == 0 {
				return nil
			}
			return copy
		}
	default:
		{
			return data
		}
	}
}

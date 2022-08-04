package client

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/weplanx/openapi/model"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Client struct {
	*client.Client
	*Option
}

type Option struct {
	Url    string
	Key    string
	Secret string
}

type OptionFunc func(x *Client)

// SetApiGateway 设置网关认证
// https://cloud.tencent.com/document/product/628/55088
func SetApiGateway(key string, secret string) OptionFunc {
	return func(x *Client) {
		x.Key = key
		x.Secret = secret
	}
}

type M = map[string]interface{}

// New 创建
func New(url string, options ...OptionFunc) (x *Client, err error) {
	x = new(Client)

	if x.Client, err = client.NewClient(
		client.WithResponseBodyStream(true),
	); err != nil {
		return
	}

	x.Option = &Option{Url: url}
	for _, v := range options {
		v(x)
	}

	return
}

// R 创建请求
func (x *Client) R(method string, path string) *OpenAPI {
	return &OpenAPI{
		Client: x,
		Method: method,
		Path:   path,
		Header: map[string]string{
			"accept": "application/json",
			"source": "apigw test",
			"x-date": time.Now().UTC().Format(http.TimeFormat),
		},
	}
}

type OpenAPI struct {
	Client *Client
	Method string
	Path   string
	Header map[string]string
	Query  url.Values
	Body   []byte
}

func (x *OpenAPI) SetHeaders(v map[string]string) *OpenAPI {
	for k, v := range v {
		x.Header[k] = v
	}
	return x
}

func (x *OpenAPI) SetQuery(v url.Values) *OpenAPI {
	x.Query = v
	return x
}

func (x *OpenAPI) SetData(v interface{}) *OpenAPI {
	x.Body, _ = sonic.Marshal(v)
	return x
}

func (x *OpenAPI) SetAuthorization() string {
	var headers []string
	var headersKVString strings.Builder
	for k, _ := range x.Header {
		if k == "Accept" {
			continue
		}
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)
	for _, v := range headers {
		headersKVString.WriteString(fmt.Sprintf("%s: %s\n", v, x.Header[v]))
	}
	accept := "application/json"
	contextMd5 := ""
	if x.Body != nil {
		hashMd5 := md5.New()
		hashMd5.Write(x.Body)
		contextMd5 = hex.EncodeToString(hashMd5.Sum(nil))
	}
	pathAndParameters := x.Path
	if x.Query != nil {
		pathAndParameters += fmt.Sprintf(`?%s`, x.Query.Encode())
	}
	signToString := fmt.Sprintf("%s%s\n%s\n\n%s\n%s",
		headersKVString.String(), x.Method, accept, contextMd5, pathAndParameters,
	)
	hmacSha256 := hmac.New(sha256.New, []byte(x.Client.Secret))
	hmacSha256.Write([]byte(signToString))
	signature := base64.StdEncoding.EncodeToString(hmacSha256.Sum(nil))
	return fmt.Sprintf(
		`hmac id="%s", algorithm="hmac-sha256", headers="%s", signature="%s"`,
		x.Client.Key, strings.Join(headers, " "), signature,
	)
}

func (x *OpenAPI) Send(ctx context.Context) (resp *protocol.Response, err error) {
	req := new(protocol.Request)
	req.SetMethod(x.Method)
	u := fmt.Sprintf(`%s%s?%s`, x.Client.Url, x.Path, x.Query.Encode())
	req.SetRequestURI(u)
	req.SetHeaders(x.Header)

	if x.Client.Key != "" && x.Client.Secret != "" {
		req.SetHeader("Authorization", x.SetAuthorization())
	}

	resp = new(protocol.Response)
	if err = x.Client.Do(ctx, req, resp); err != nil {
		return
	}

	return
}

// Ping 测试
func (x *Client) Ping(ctx context.Context) (result M, err error) {
	var resp *protocol.Response
	if resp, err = x.R("GET", "/").Send(ctx); err != nil {
		return
	}
	result = make(M)
	if err = sonic.Unmarshal(resp.Body(), &result); err != nil {
		return
	}
	return
}

// GetIp 获取 Ip
func (x *Client) GetIp(ctx context.Context, ip string) (data M, err error) {
	query := make(url.Values)
	query.Set("value", ip)
	var resp *protocol.Response
	if resp, err = x.R("GET", "/ip").
		SetQuery(query).
		Send(ctx); err != nil {
		return
	}
	data = make(M)
	if err = sonic.Unmarshal(resp.Body(), &data); err != nil {
		return
	}
	return
}

func (x *Client) GetCountries(ctx context.Context, fields []string) (data []model.Country, err error) {
	query := make(url.Values)
	query.Set("fields", strings.Join(fields, ","))
	var resp *protocol.Response
	if resp, err = x.R("GET", "/geo/countries").
		SetQuery(query).
		Send(ctx); err != nil {
		return
	}
	data = make([]model.Country, 0)
	if err = sonic.Unmarshal(resp.Body(), &data); err != nil {
		return
	}
	return
}

func (x *Client) GetStates(ctx context.Context, country string, fields []string) (data []model.State, err error) {
	query := make(url.Values)
	query.Set("country", country)
	query.Set("fields", strings.Join(fields, ","))
	var resp *protocol.Response
	if resp, err = x.R("GET", "/geo/states").
		SetQuery(query).
		Send(ctx); err != nil {
		return
	}
	data = make([]model.State, 0)
	if err = sonic.Unmarshal(resp.Body(), &data); err != nil {
		return
	}
	return
}

func (x *Client) GetCities(ctx context.Context, country string, state string, fields []string) (data []model.City, err error) {
	query := make(url.Values)
	query.Set("country", country)
	query.Set("state", state)
	query.Set("fields", strings.Join(fields, ","))
	var resp *protocol.Response
	if resp, err = x.R("GET", "/geo/cities").
		SetQuery(query).
		Send(ctx); err != nil {
		return
	}
	data = make([]model.City, 0)
	if err = sonic.Unmarshal(resp.Body(), &data); err != nil {
		return
	}
	return
}

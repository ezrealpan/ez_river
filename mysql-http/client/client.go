package apatodon

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/juju/errors"
)

const (
	attempts = 3
	sleep    = 1 * time.Second
	authErr  = 16
)

// ErrNoAuth ...
var ErrNoAuth = errors.New("no auth")

// Client is the client to communicate with apatodon.
type Client struct {
	Protocol string
	Addr     string
	User     string
	Password string
	LoginURL string
	Token    string

	c *http.Client
}

// ClientConfig is the configuration for the client.
type ClientConfig struct {
	HTTPS    bool
	Addr     string
	User     string
	Password string
	LoginURL string
}

// NewClient creates the Cient with configuration.
func NewClient(conf *ClientConfig) *Client {
	c := new(Client)

	c.Addr = conf.Addr
	c.User = conf.User
	c.Password = conf.Password
	c.LoginURL = conf.LoginURL

	if conf.HTTPS {
		c.Protocol = "https"
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c.c = &http.Client{Transport: tr}
	} else {
		c.Protocol = "http"
		c.c = &http.Client{}
	}
	_, err := c.Login()
	if err != nil {
		log.Fatal("登录失败:", err)
	}
	return c
}

// ResponseItem is the ES item in the response.
type ResponseItem struct {
	ID      string                 `json:"_id"`
	Index   string                 `json:"_index"`
	Type    string                 `json:"_type"`
	Version int                    `json:"_version"`
	Found   bool                   `json:"found"`
	Source  map[string]interface{} `json:"_source"`
}

// Response is the ES response
type Response struct {
	Code    int
	Message string
	Data    interface{}
}

// BulkRequest is used to send multi request in batch.
type BulkRequest struct {
	Action   string
	Index    string
	Type     string
	ID       string
	Parent   string
	Pipeline string

	Data map[string]interface{}
}

// Login ...
func (c *Client) Login() (*Response, error) {
	reqURL := fmt.Sprintf("%s://%s/%s", c.Protocol, c.Addr, c.LoginURL)
	data := map[string]interface{}{
		"username": c.User,
		"password": c.Password,
	}
	r, err := c.Do("post", reqURL, data)
	if err != nil {
		return r, err
	}
	token, right := r.Data.(string)
	if !right {
		return r, fmt.Errorf("数据格式错误")
	}
	c.Token = token
	return r, err
}

// DoRequest sends a request with body to apatodon.
func (c *Client) DoRequest(method string, url string, body *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "JWT "+c.Token)
	if err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := c.c.Do(req)

	return resp, err
}

// Do sends the request with body to apatodon.
func (c *Client) Do(method string, url string, body map[string]interface{}) (*Response, error) {
	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(bodyData)
	if body == nil {
		buf = bytes.NewBuffer(nil)
	}

	resp, err := c.DoRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ret := new(Response)

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

// Get gets the item by id.
func (c *Client) Get(index string, docType string, id string) error {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/%s", c.Protocol, c.Addr, index, docType, id)
	r := new(Response)
	if err := retry(attempts, sleep, func() error {
		var err error
		//1. 业务代码
		r, err = c.Do("GET", reqURL, nil)
		if err != nil {
			return stop{err} //不需要retry 并且有错误信息返回
		}

		//2. 如果不是 16的认证错误 直接跳出
		if r.Code != authErr {
			return nil //不需要retry 并且没有错误
		}

		//3. 补充token
		_, err = c.Login()
		if err != nil {
			return stop{err}
		}
		return ErrNoAuth
	}); err != nil {
		return err
	}

	if r.Code != 0 {
		return fmt.Errorf("Put failed, code: %v, message: %v", r.Code, r.Message)
	}
	return nil
}

// Update creates or updates the data
func (c *Client) Update(index string, docType string, id string, data map[string]interface{}) error {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/%s", c.Protocol, c.Addr, index, docType, id)
	r := new(Response)
	if err := retry(attempts, sleep, func() error {
		var err error
		//1. 业务代码
		r, err = c.Do("PUT", reqURL, data)
		if err != nil {
			return stop{err} //不需要retry 并且有错误信息返回
		}

		//2. 如果不是 16的认证错误 直接跳出
		if r.Code != authErr {
			return nil //不需要retry 并且没有错误
		}

		//3. 补充token
		_, err = c.Login()
		if err != nil {
			return stop{err}
		}
		return ErrNoAuth
	}); err != nil {
		return err
	}

	if r.Code != 0 {
		return fmt.Errorf("Put failed, code: %v, message: %v", r.Code, r.Message)
	}
	return nil
}

// Delete deletes the item by id.
func (c *Client) Delete(index string, docType string, id string) error {
	reqURL := fmt.Sprintf("%s://%s/%s/%s/%s", c.Protocol, c.Addr, index, docType, id)
	r := new(Response)
	if err := retry(attempts, sleep, func() error {
		var err error
		//1. 业务代码
		r, err = c.Do("DELETE", reqURL, nil)
		if err != nil {
			return stop{err} //不需要retry 并且有错误信息返回
		}

		//2. 如果不是 16的认证错误 直接跳出
		if r.Code != authErr {
			return nil //不需要retry 并且没有错误
		}

		//3. 补充token
		_, err = c.Login()
		if err != nil {
			return stop{err}
		}
		return ErrNoAuth
	}); err != nil {
		return err
	}

	if r.Code != 0 {
		return fmt.Errorf("Delete failed, code: %v, message: %v", r.Code, r.Message)
	}
	return nil
}

// Create data
func (c *Client) Create(index string, docType string, data map[string]interface{}) error {
	reqURL := fmt.Sprintf("%s://%s/%s/%s", c.Protocol, c.Addr, index, docType)
	r := new(Response)
	if err := retry(attempts, sleep, func() error {
		var err error
		//1. 业务代码
		r, err = c.Do("POST", reqURL, data)
		if err != nil {
			return stop{err} //不需要retry 并且有错误信息返回
		}

		//2. 如果不是 16的认证错误 直接跳出
		if r.Code != authErr {
			return nil //不需要retry 并且没有错误
		}

		//3. 补充token
		_, err = c.Login()
		if err != nil {
			return stop{err}
		}
		return ErrNoAuth
	}); err != nil {
		return err
	}

	if r.Code != 0 {
		return fmt.Errorf("Post failed, code: %v, message: %v", r.Code, r.Message)
	}
	return nil
}

func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd

			time.Sleep(sleep)
			return retry(attempts, sleep, f)
		}
		return err
	}

	return nil
}

type stop struct {
	error
}

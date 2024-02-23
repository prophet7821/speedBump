# SpeedBump

[//]: # ([![Go Report Card]&#40;https://goreportcard.com/badge/github.com/cloudflare/speedbump&#41;]&#40;https://goreportcard.com/report/github.com/cloudflare/speedbump&#41;)

[//]: # ([![GoDoc]&#40;https://godoc.org/github.com/cloudflare/speedbump?status.svg&#41;]&#40;https://godoc.org/github.com/cloudflare/speedbump&#41;)

[//]: # ([![License]&#40;https://img.shields.io/badge/License-BSD%203--Clause-blue.svg&#41;]&#40;)

SpeedBump is a powerful and flexible rate limiting library for Go, designed to protect your web applications and APIs
from overuse and to ensure equitable resource access across all users. At its core, SpeedBump utilizes Cloudflare's
sliding window counter algorithm, offering a sophisticated approach to rate limiting that balances fairness and
efficiency. This method ensures that request limits are enforced smoothly over time, preventing bursts of traffic from
unfairly consuming resources

## Installation

```bash
go get github.com/prophet7821/speedbump.git
````

## Usage

SpeedBump provides a suite of functionalities to easily integrate rate limiting into your Go applications. Below is a
table summarizing the available functions:

| Function                | Description                                               | Example                                                                                                                                                                                                                                                                             |
|-------------------------|-----------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Limit`                 | Applies rate limits with customizable options.            | <pre><code> router.Use(<br/>speedBump.Limit(<br/>100, <br/>1*time.Minute))</pre></code>                                                                                                                                                                                             |
| `LimitAll`              | Applies a global rate limit across all requests.          | <pre><code>router.Use(<br/>speedBump.LimitAll(<br/>100, <br/>1*time.Minute))</pre></code>                                                                                                                                                                                           |
| `LimitByIP`             | Limits requests based on the client's IP address.         | <pre><code>router.Use(<br/>speedBump.LimitByIP(<br/>100, <br/>1*time.Minute))</pre></code>                                                                                                                                                                                          |
| `WithKeyFuncs`          | Allows setting custom key functions for rate limiting.    | <pre><code>router.Use(<br/>speedBump.Limit(<br/>100,<br/>1*time.Minute,<br/>speedBump.WithKeyFuncs(speedBump.KeyByIP, func (r *http.Request) (string, error) {<br/>return r.Header.Get("X-Custom-Header"), nil<br/>}),))</pre></code>                                               |
| `KeyByIP/EndPoint`      | Provides built-in key functions for identifying requests. | <pre><code>router.Use(<br/>speedBump.Limit(<br/>100,<br/>1*time.Minute,<br/>speedBump.WithKeyFuncs(speedBump.KeyByIP, speedBump.KeyByEndPoint)<br/>))</pre></code>                                                                                                                  |
| `WithLimitHandler`      | Customizes the response for rate-limited requests.        | <pre><code>router.Use(<br/>speedBump.Limit(<br/>100,<br/>1*time.Minute,<br/>speedBump.WithLimitHandler(<br/>func(w http.ResponseWriter, r *http.Request) {<br/>w.WriteHeader(http.StatusTooManyRequests)<br/>w.Write([]byte("Custom limit exceeded message"))<br/>})))</pre></code> |
| `WithLimitCounter`      | Enables the use of a custom limit counter.                | _See below for integrating with Redis_                                                                                                                                                                                                                                              |
| `WithRedisLimitCounter` | Integrates Redis for distributed rate limiting.           | <pre><code>router.Use(<br/>speedBump.WithRedisLimitCounter(<br/>&speedBump.Config{<br/>Host: "localhost",<br/>Port: 6379,Password: "", <br/>// Optional: Your Redis password, if any.DBIndex: 0, <br/>// Optional: The Redis database index.<br/>}))</pre></code>                   |

## Contributing

We welcome contributions to the SpeedBump project! Whether it's adding new features, fixing bugs, improving
documentation, or sharing feedback, your collaboration is highly appreciated. Please feel free to submit pull requests,
report issues, or suggest improvements. Let's work together to make SpeedBump even better for the community.


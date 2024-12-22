package hf

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chatgpt-adapter/core/common"
	"chatgpt-adapter/core/logger"
	"github.com/bincooo/emit.io"
	"github.com/gin-gonic/gin"
	"github.com/iocgo/sdk/env"
)

const (
	negative  = "(text:1.3), (strip cartoon:1.3), out of focus, fewer digits, cropped, signature, watermark"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0"
)

func Ox0(ctx *gin.Context, env *env.Environment, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxies = env.GetString("server.proxied")
		baseUrl = "https://prodia-fast-stable-diffusion.hf.space"
		domain  = env.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	fn := []int{0, 15}
	data := []interface{}{
		message + ", {{{{by famous artist}}}, beautiful, 4k",
		negative,
		model,
		25,
		samples,
		10,
		1024,
		1024,
		-1,
	}
	response, err := emit.ClientBuilder(common.HTTPClient).
		Proxies(proxies).
		Context(ctx.Request.Context()).
		POST(baseUrl+"/queue/join").
		JSONHeader().
		Header("User-Agent", userAgent).
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(common.HTTPClient).
		Proxies(proxies).
		Context(ctx.Request.Context()).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("User-Agent", userAgent).
		DoS(http.StatusOK)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data
		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}
		result := d[0].(map[string]interface{})
		value = result["url"].(string)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox1(ctx *gin.Context, env *env.Environment, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxied = env.GetString("server.proxied")
		baseUrl = "wss://prodia-sdxl-stable-diffusion-xl.hf.space"
		domain  = env.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, response, err := emit.SocketBuilder(common.HTTPClient).
		Proxies(proxied).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), conn)
	if err != nil {
		return
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     0,
			"session_hash": hash,
			"data": []interface{}{
				message + ", {{{{by famous artist}}}, beautiful, 4k",
				negative,
				model,
				25,
				samples,
				10,
				1024,
				1024,
				-1,
			},
		}
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		var file string
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if file, err = common.SaveBase64(d[0].(string), "png"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox2(ctx *gin.Context, env *env.Environment, model, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxied = env.GetString("server.proxied")
		baseUrl = "https://mukaist-dalle-4k.hf.space"
	)

	if u := env.GetString("hf.dalle-4k.base-url"); u != "" {
		baseUrl = u
	}

	fn := []int{3, 6}
	data := []interface{}{
		message,
		negative,
		true,
		model,
		30,
		1024,
		1024,
		6,
		true,
	}
	fn, data, err = bindAttr(env, "dalle-4k", fn, data, message, negative, "", model, -1)
	response, err := emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		POST(baseUrl+"/queue/join").
		JSONHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).
		DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()
	response, err = emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}
		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		value = v["image"].(map[string]interface{})["url"].(string)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func Ox3(ctx *gin.Context, env *env.Environment, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxied = env.GetString("server.proxied")
		domain  = env.GetString("domain")
		baseUrl = "https://ehristoforu-dalle-3-xl-lora-v2.hf.space"
		r       = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	if u := env.GetString("hf.dalle-3-xl.base-url"); u != "" {
		baseUrl = u
	}

	fn := []int{3, 6}
	data := []interface{}{
		message + ", {{{{by famous artist}}}, beautiful, 4k",
		negative + ", extra limb, missing limb, floating limbs, (mutated hands and fingers:1.4), disconnected limbs, mutation, mutated, ugly, disgusting, blurry, amputation",
		true,
		r.Intn(51206501) + 1100000000,
		1024,
		1024,
		12,
		true,
	}
	fn, data, err = bindAttr(env, "dalle-3-xl", fn, data, message, negative, "", "", r.Intn(51206501)+1100000000)
	response, err := emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		POST(baseUrl+"/queue/join").
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept-Language", "en-US,en;q=0.9").
		JSONHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return "", err
	}

	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept", "text/event-stream").
		Header("Accept-Language", "en-US,en;q=0.9").
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return "", err
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		info, ok := v["image"].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		// 锁环境了，只能先下载下来
		value, err = common.Download(common.HTTPClient, proxied, info["url"].(string), "png", map[string]string{
			// "User-Agent":      userAgent,
			// "Accept-Language": "en-US,en;q=0.9",
			"Origin":  "https://huggingface.co",
			"Referer": baseUrl + "/?__theme=light",
		})
		if err != nil {
			c.Failed(fmt.Errorf("image download failed: %v", err))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, value)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

// 潦草漫画的风格
func Ox4(ctx *gin.Context, env *env.Environment, model, samples, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxied = ctx.GetString("server.proxied")
		domain  = env.GetString("domain")
		baseUrl = "https://cagliostrolab-animagine-xl-3-1.hf.space"
		r       = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	if u := env.GetString("hf.animagine-xl-3.1.base-url"); u != "" {
		baseUrl = u
	}

	fn := []int{5, 49}
	data := []interface{}{
		message,
		negative,
		r.Intn(1490935504) + 9068457,
		1024,
		1024,
		7,
		35,
		samples,
		"1024 x 1024",
		model,
		"Heavy v3.1",
		false,
		0.55,
		1.5,
		true,
	}
	fn, data, err = bindAttr(env, "animagine-xl-3.1", fn, data, message, negative, samples, model, r.Intn(1490935504)+9068457)
	response, err := emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		POST(baseUrl+"/queue/join").
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept-Language", "en-US,en;q=0.9").
		JSONHeader().
		Body(map[string]interface{}{
			"data":         data,
			"fn_index":     fn[0],
			"trigger_id":   fn[1],
			"session_hash": hash,
		}).DoC(emit.Status(http.StatusOK), emit.IsJSON)
	if err != nil {
		return "", err
	}
	logger.Info(emit.TextResponse(response))
	_ = response.Body.Close()

	response, err = emit.ClientBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		GET(baseUrl+"/queue/data").
		Query("session_hash", hash).
		Header("Origin", baseUrl).
		Header("Referer", baseUrl+"/?__theme=light").
		Header("User-Agent", userAgent).
		Header("Accept", "text/event-stream").
		Header("Accept-Language", "en-US,en;q=0.9").
		DoC(emit.Status(http.StatusOK), emit.IsSTREAM)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	c, err := emit.NewGio(ctx.Request.Context(), response)
	if err != nil {
		return "", err
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		if len(values) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		v, ok := values[0].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		info, ok := v["image"].(map[string]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		// 锁环境了，只能先下载下来
		value, err = common.Download(common.HTTPClient, proxied, info["url"].(string), "png", map[string]string{
			// "User-Agent":      userAgent,
			// "Accept-Language": "en-US,en;q=0.9",
			"Origin":  "https://huggingface.co",
			"Referer": baseUrl + "/?__theme=light",
		})
		if err != nil {
			c.Failed(fmt.Errorf("image download failed: %v", err))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, value)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func google(ctx *gin.Context, env *env.Environment, model, message string) (value string, err error) {
	var (
		hash    = emit.GioHash()
		proxied = ctx.GetString("server.proxied")
		baseUrl = "wss://google-sdxl.hf.space"
		domain  = env.GetString("domain")
	)

	if domain == "" {
		domain = fmt.Sprintf("http://127.0.0.1:%d", ctx.GetInt("port"))
	}

	conn, response, err := emit.SocketBuilder(common.HTTPClient).
		Proxies(proxied).
		Context(ctx.Request.Context()).
		URL(baseUrl + "/queue/join").
		DoS(http.StatusSwitchingProtocols)
	if err != nil {
		return
	}
	defer response.Body.Close()

	c, err := emit.NewGio(ctx.Request.Context(), conn)
	if err != nil {
		return
	}

	c.Event("*", func(j emit.JoinEvent) (_ interface{}) {
		logger.Debugf("event: %s", j.InitialBytes)
		return
	})

	c.Event("send_hash", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     2,
			"session_hash": hash,
		}
	})

	c.Event("send_data", func(j emit.JoinEvent) interface{} {
		return map[string]interface{}{
			"fn_index":     2,
			"session_hash": hash,
			"data": []interface{}{
				message + ", {{{{by famous artist}}}, beautiful, 4k",
				negative,
				25,
				model,
			},
		}
	})

	c.Event("process_completed", func(j emit.JoinEvent) (_ interface{}) {
		var file string
		d := j.Output.Data

		if len(d) == 0 {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		values, ok := d[0].([]interface{})
		if !ok {
			c.Failed(fmt.Errorf("image generate failed: %s", j.InitialBytes))
			return
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if file, err = common.SaveBase64(values[r.Intn(len(values))].(string), "jpg"); err != nil {
			c.Failed(fmt.Errorf("image save failed: %s", j.InitialBytes))
			return
		}

		value = fmt.Sprintf("%s/file/%s", domain, file)
		return
	})

	err = c.Do()
	if err == nil && value == "" {
		err = fmt.Errorf("image generate failed")
	}
	return
}

func bindAttr(env *env.Environment, key string, fn []int, data []interface{}, message, negative, sampler, style string, seed int) ([]int, []interface{}, error) {
	slice := env.GetIntSlice("hf." + key + ".fn")
	if len(slice) >= 2 {
		fn = slice
	}
	dataStr := env.GetString("hf." + key + ".data")
	if dataStr != "" {
		dataStr = strings.ReplaceAll(dataStr, "{{prompt}}", message)
		dataStr = strings.ReplaceAll(dataStr, "{{negative_prompt}}", negative)
		dataStr = strings.ReplaceAll(dataStr, "{{sampler}}", sampler)
		dataStr = strings.ReplaceAll(dataStr, "{{style}}", style)
		dataStr = strings.ReplaceAll(dataStr, "{{seed}}", strconv.Itoa(seed))
		err := json.Unmarshal([]byte(dataStr), &data)
		if err != nil {
			return nil, nil, err
		}
	}
	return fn, data, nil
}

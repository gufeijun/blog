package httpd

type Header map[string][]string

func (h Header) Add(key, value string) {
	h[key] = append(h[key], value)
}

func (h Header) Set(key, value string) {
	h[key] = []string{value}
}

//Get获取key对应的第一个value，如果不存在对应的key，则return ""
func (h Header) Get(key string) string {
	if value, ok := h[key]; ok && len(value) > 0 {
		return value[0]
	} else {
		return ""
	}
}

func (h Header) Del(key string) {
	delete(h, key)
}

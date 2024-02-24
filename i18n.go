// Copyright 2015 Eryx <evorui аt gmаil dοt cοm>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsrv

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	i18nMut       sync.RWMutex
	i18n          = map[string]string{}
	i18nDefLocale = "en"
	i18nRegPath   = regexp.MustCompile("/+")
)

type i18nConfig struct {
	Locale string `json:"locale" toml:"locale"`
	Data   []i18nConfigItem
}

type i18nConfigItem struct {
	Key string `json:"key" toml:"key"`
	Val string `json:"val" toml:"val"`
}

func I18nFilter(c *Controller) {

	if v, e := c.Request.Cookie(c.service.Config.CookieKeyLocale); e == nil {
		c.Request.Locale = v.Value
	} else if len(c.Request.acceptLanguage) > 0 {
		c.Request.Locale = c.Request.acceptLanguage[0].Language
	} else {
		c.Request.Locale = i18nDefLocale
	}

	c.Data["LANG"] = c.Request.Locale
}

func i18nLoadMessages(file string) {

	i18nMut.Lock()
	defer i18nMut.Unlock()

	var cfg i18nConfig

	str, err := i18nFsFileGetRead(file)
	if err != nil {
		defaultLogger.Warnf("httpsrv/lang load file (%s) err %s", file, err.Error())
		return
	}

	if err := jsonDecode([]byte(str), &cfg); err != nil {
		defaultLogger.Warnf("httpsrv/lang setup err %s", err.Error())
		return
	}

	cfg.Locale = strings.Replace(cfg.Locale, "_", "-", 1)

	for _, v := range cfg.Data {

		key := strings.ToLower(cfg.Locale + "." + v.Key)

		if v2, ok := i18n[key]; !ok || v2 != v.Val {
			i18n[key] = v.Val
		}
	}
}

func i18nTranslate(locale, msg string, args ...interface{}) string {

	key := strings.ToLower(locale + "." + msg)
	keydef := strings.ToLower(i18nDefLocale + "." + msg)

	i18nMut.RLock()

	if v, ok := i18n[key]; ok {
		msg = v
	} else if v, ok := i18n[keydef]; ok {
		msg = v
	}

	i18nMut.RUnlock()

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

func i18nFsFileGetRead(path string) (string, error) {

	path = "/" + strings.Trim(i18nRegPath.ReplaceAllString(path, "/"), "/")

	i18nMut.Lock()
	defer i18nMut.Unlock()

	if st, err := os.Stat(path); err != nil || os.IsNotExist(err) {
		return "", errors.New("File Not Found")
	} else if st.Size() > (10 << 20) {
		return "", errors.New("File size is too large")
	}

	ctn, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.New("File Can Not Readable")
	}

	return string(ctn), nil
}

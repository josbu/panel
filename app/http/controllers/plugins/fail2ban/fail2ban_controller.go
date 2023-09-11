package fail2ban

import (
	"regexp"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"panel/app/http/controllers"
	"panel/app/models"
	"panel/app/services"
	"panel/pkg/tools"
)

type Fail2banController struct {
	website services.Website
}

func NewFail2banController() *Fail2banController {
	return &Fail2banController{
		website: services.NewWebsiteImpl(),
	}
}

type Jail struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	LogPath  string `json:"log_path"`
	MaxRetry int    `json:"max_retry"`
	FindTime int    `json:"find_time"`
	BanTime  int    `json:"ban_time"`
}

// Status 获取运行状态
func (c *Fail2banController) Status(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	status := tools.Exec("systemctl status fail2ban | grep Active | grep -v grep | awk '{print $2}'")
	if len(status) == 0 {
		return controllers.Error(ctx, http.StatusInternalServerError, "获取服务运行状态失败")
	}

	if status == "active" {
		return controllers.Success(ctx, true)
	} else {
		return controllers.Success(ctx, false)
	}
}

// Reload 重载配置
func (c *Fail2banController) Reload(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	tools.Exec("systemctl reload fail2ban")
	status := tools.Exec("systemctl status fail2ban | grep Active | grep -v grep | awk '{print $2}'")
	if len(status) == 0 {
		return controllers.Error(ctx, http.StatusInternalServerError, "获取服务运行状态失败")
	}

	if status == "active" {
		return controllers.Success(ctx, true)
	} else {
		return controllers.Success(ctx, false)
	}
}

// Restart 重启服务
func (c *Fail2banController) Restart(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	tools.Exec("systemctl restart fail2ban")
	status := tools.Exec("systemctl status fail2ban | grep Active | grep -v grep | awk '{print $2}'")
	if len(status) == 0 {
		return controllers.Error(ctx, http.StatusInternalServerError, "获取服务运行状态失败")
	}

	if status == "active" {
		return controllers.Success(ctx, true)
	} else {
		return controllers.Success(ctx, false)
	}
}

// Start 启动服务
func (c *Fail2banController) Start(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	tools.Exec("systemctl start fail2ban")
	status := tools.Exec("systemctl status fail2ban | grep Active | grep -v grep | awk '{print $2}'")
	if len(status) == 0 {
		return controllers.Error(ctx, http.StatusInternalServerError, "获取服务运行状态失败")
	}

	if status == "active" {
		return controllers.Success(ctx, true)
	} else {
		return controllers.Success(ctx, false)
	}
}

// Stop 停止服务
func (c *Fail2banController) Stop(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	tools.Exec("systemctl stop fail2ban")
	status := tools.Exec("systemctl status fail2ban | grep Active | grep -v grep | awk '{print $2}'")
	if len(status) == 0 {
		return controllers.Error(ctx, http.StatusInternalServerError, "获取服务运行状态失败")
	}

	if status != "active" {
		return controllers.Success(ctx, true)
	} else {
		return controllers.Success(ctx, false)
	}
}

// List 所有 Fail2ban 规则
func (c *Fail2banController) List(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	page := ctx.Request().QueryInt("page", 1)
	limit := ctx.Request().QueryInt("limit", 10)
	raw := tools.Read("/etc/fail2ban/jail.local")
	if len(raw) == 0 {
		return controllers.Error(ctx, http.StatusBadRequest, "Fail2ban 规则为空")
	}

	jailList := regexp.MustCompile(`\[(.*?)]`).FindAllStringSubmatch(raw, -1)
	if len(jailList) == 0 {
		return controllers.Error(ctx, http.StatusBadRequest, "Fail2ban 规则为空")
	}

	var jails []Jail
	for i, jail := range jailList {
		if i == 0 {
			continue
		}

		jailName := jail[1]
		jailRaw := tools.Cut(raw, "# "+jailName+"-START", "# "+jailName+"-END")
		if len(jailRaw) == 0 {
			continue
		}
		jailEnabled := strings.Contains(jailRaw, "enabled = true")
		jailLogPath := regexp.MustCompile(`logpath = (.*)`).FindStringSubmatch(jailRaw)
		jailMaxRetry := regexp.MustCompile(`maxretry = (.*)`).FindStringSubmatch(jailRaw)
		jailFindTime := regexp.MustCompile(`findtime = (.*)`).FindStringSubmatch(jailRaw)
		jailBanTime := regexp.MustCompile(`bantime = (.*)`).FindStringSubmatch(jailRaw)

		jails = append(jails, Jail{
			Name:     jailName,
			Enabled:  jailEnabled,
			LogPath:  jailLogPath[1],
			MaxRetry: cast.ToInt(jailMaxRetry[1]),
			FindTime: cast.ToInt(jailFindTime[1]),
			BanTime:  cast.ToInt(jailBanTime[1]),
		})
	}

	startIndex := (page - 1) * limit
	endIndex := page * limit
	if startIndex > len(jails) {
		return controllers.Success(ctx, http.Json{
			"total": 0,
			"items": []Jail{},
		})
	}
	if endIndex > len(jails) {
		endIndex = len(jails)
	}
	pagedJails := jails[startIndex:endIndex]

	return controllers.Success(ctx, http.Json{
		"total": len(jails),
		"items": pagedJails,
	})
}

// Add 添加 Fail2ban 规则
func (c *Fail2banController) Add(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	validator, err := ctx.Request().Validate(map[string]string{
		"name":         "required",
		"type":         "required|in:website,service",
		"maxretry":     "required",
		"findtime":     "required",
		"bantime":      "required",
		"website_mode": "required_if:type,website",
		"website_path": "required_if:type,website",
	})
	if err != nil {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, err.Error())
	}
	if validator.Fails() {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, validator.Errors().One())
	}

	jailName := ctx.Request().Input("name")
	jailType := ctx.Request().Input("type")
	jailMaxRetry := ctx.Request().Input("maxretry")
	jailFindTime := ctx.Request().Input("findtime")
	jailBanTime := ctx.Request().Input("bantime")
	jailWebsiteMode := ctx.Request().Input("website_mode")
	jailWebsitePath := ctx.Request().Input("website_path")

	raw := tools.Read("/etc/fail2ban/jail.local")
	if strings.Contains(raw, "["+jailName+"]") || (strings.Contains(raw, "["+jailName+"]"+"-cc") && jailWebsiteMode == "cc") || (strings.Contains(raw, "["+jailName+"]"+"-path") && jailWebsiteMode == "path") {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, "规则已存在")
	}

	switch jailType {
	case "website":
		var website models.Website
		err := facades.Orm().Query().Where("name", jailName).FirstOrFail(&website)
		if err != nil {
			return controllers.Error(ctx, http.StatusUnprocessableEntity, "网站不存在")
		}
		config, err := c.website.GetConfig(int(website.ID))
		if err != nil {
			return controllers.Error(ctx, http.StatusUnprocessableEntity, "获取网站配置失败")
		}
		var ports string
		for _, port := range config.Ports {
			if len(strings.Split(port, " ")) > 1 {
				ports += strings.Split(port, " ")[0] + ","
			} else {
				ports += port + ","
			}
		}

		rule := `
# ` + jailName + `-` + jailWebsiteMode + `-START
[` + jailName + `-` + jailWebsiteMode + `]
enabled = true
filter = haozi-` + jailName + `-` + jailWebsiteMode + `
port = ` + ports + `
maxretry = ` + jailMaxRetry + `
findtime = ` + jailFindTime + `
bantime = ` + jailBanTime + `
action = %(action_mwl)s
logpath = /www/wwwlogs/` + website.Name + `.log
# ` + jailName + `-` + jailWebsiteMode + `-END
`
		raw += rule
		tools.Write("/etc/fail2ban/jail.local", raw, 0644)

		var filter string
		if jailWebsiteMode == "cc" {
			filter = `
[Definition]
failregex = ^<HOST>\s-.*HTTP/.*$
ignoreregex =
`
		} else {
			filter = `
[Definition]
failregex = ^<HOST>\s-.*\s` + jailWebsitePath + `.*HTTP/.*$
ignoreregex =
`
		}
		tools.Write("/etc/fail2ban/filter.d/haozi-"+jailName+"-"+jailWebsiteMode+".conf", filter, 0644)

	case "service":
		var logPath string
		var filter string
		var port string
		switch jailName {
		case "ssh":
			if tools.IsDebian() {
				logPath = "/var/log/auth.log"
			} else {
				logPath = "/var/log/secure"
			}
			filter = "sshd"
			port = tools.Exec("cat /etc/ssh/sshd_config | grep 'Port ' | awk '{print $2}'")
		case "mysql":
			logPath = "/www/server/mysql/mysql-error.log"
			filter = "mysqld-auth"
			port = tools.Exec("cat /www/server/mysql/conf/my.cnf | grep 'port' | head -n 1 | awk '{print $3}'")
		case "pure-ftpd":
			logPath = "/var/log/messages"
			filter = "pure-ftpd"
			port = tools.Exec(`cat /www/server/pure-ftpd/etc/pure-ftpd.conf | grep "Bind" | awk '{print $2}' | awk -F "," '{print $2}'`)
		default:
			return controllers.Error(ctx, http.StatusUnprocessableEntity, "未知服务")
		}
		if len(port) == 0 {
			return controllers.Error(ctx, http.StatusUnprocessableEntity, "获取服务端口失败，请检查是否安装")
		}

		rule := `
# ` + jailName + `-START
[` + jailName + `]
enabled = true
filter = ` + filter + `
port = ` + port + `
maxretry = ` + jailMaxRetry + `
findtime = ` + jailFindTime + `
bantime = ` + jailBanTime + `
action = %(action_mwl)s
logpath = ` + logPath + `
# ` + jailName + `-END
`
		raw += rule
		tools.Write("/etc/fail2ban/jail.local", raw, 0644)
	}

	tools.Exec("fail2ban-client reload")
	return controllers.Success(ctx, nil)
}

// Delete 删除规则
func (c *Fail2banController) Delete(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	jailName := ctx.Request().Input("name")
	raw := tools.Read("/etc/fail2ban/jail.local")
	if !strings.Contains(raw, "["+jailName+"]") {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, "规则不存在")
	}

	rule := tools.Cut(raw, "# "+jailName+"-START", "# "+jailName+"-END")
	raw = strings.Replace(raw, "\n# "+jailName+"-START"+rule+"# "+jailName+"-END", "", -1)
	raw = strings.TrimSpace(raw)
	tools.Write("/etc/fail2ban/jail.local", raw, 0644)

	tools.Exec("fail2ban-client reload")
	return controllers.Success(ctx, nil)
}

// BanList 获取封禁列表
func (c *Fail2banController) BanList(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	name := ctx.Request().Query("name")
	if len(name) == 0 {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, "缺少参数")
	}

	currentlyBan := tools.Exec(`fail2ban-client status ` + name + ` | grep "Currently banned" | awk '{print $4}'`)
	totalBan := tools.Exec(`fail2ban-client status ` + name + ` | grep "Total banned" | awk '{print $4}'`)
	bannedIp := tools.Exec(`fail2ban-client status ` + name + ` | grep "Banned IP list" | awk -F ":" '{print $2}'`)
	bannedIpList := strings.Split(bannedIp, " ")

	var list []map[string]string
	for _, ip := range bannedIpList {
		if len(ip) > 0 {
			list = append(list, map[string]string{
				"name": name,
				"ip":   ip,
			})
		}
	}

	return controllers.Success(ctx, http.Json{
		"currentlyBan": currentlyBan,
		"totalBan":     totalBan,
		"bannedIpList": list,
	})
}

// Unban 解封
func (c *Fail2banController) Unban(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	name := ctx.Request().Input("name")
	ip := ctx.Request().Input("ip")
	if len(name) == 0 || len(ip) == 0 {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, "缺少参数")
	}

	tools.Exec("fail2ban-client set " + name + " unbanip " + ip)
	return controllers.Success(ctx, nil)
}

// SetWhiteList 设置白名单
func (c *Fail2banController) SetWhiteList(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	ip := ctx.Request().Input("ip")
	if len(ip) == 0 {
		return controllers.Error(ctx, http.StatusUnprocessableEntity, "缺少参数")
	}

	raw := tools.Read("/etc/fail2ban/jail.local")
	// 正则替换
	reg := regexp.MustCompile(`ignoreip\s*=\s*.*\n`)
	if reg.MatchString(raw) {
		raw = reg.ReplaceAllString(raw, "ignoreip = "+ip+"\n")
	} else {
		return controllers.Error(ctx, http.StatusInternalServerError, "解析Fail2ban规则失败，Fail2ban可能已损坏")
	}

	tools.Write("/etc/fail2ban/jail.local", raw, 0644)
	tools.Exec("fail2ban-client reload")
	return controllers.Success(ctx, nil)
}

// GetWhiteList 获取白名单
func (c *Fail2banController) GetWhiteList(ctx http.Context) http.Response {
	if !controllers.Check(ctx, "fail2ban") {
		return nil
	}

	raw := tools.Read("/etc/fail2ban/jail.local")
	reg := regexp.MustCompile(`ignoreip\s*=\s*(.*)\n`)
	if reg.MatchString(raw) {
		ignoreIp := reg.FindStringSubmatch(raw)[1]
		return controllers.Success(ctx, ignoreIp)
	} else {
		return controllers.Error(ctx, http.StatusInternalServerError, "解析Fail2ban规则失败，Fail2ban可能已损坏")
	}
}

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gookit/color"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"
	"github.com/spf13/cast"

	"panel/app/models"
	"panel/app/services"
	"panel/pkg/tools"
)

type Panel struct {
}

// Signature The name and signature of the console command.
func (receiver *Panel) Signature() string {
	return "panel"
}

// Description The console command description.
func (receiver *Panel) Description() string {
	return "[面板] 命令行"
}

// Extend The console command extend.
func (receiver *Panel) Extend() command.Extend {
	return command.Extend{
		Category: "panel",
	}
}

// Handle Execute the console command.
func (receiver *Panel) Handle(ctx console.Context) error {
	action := ctx.Argument(0)
	arg1 := ctx.Argument(1)
	arg2 := ctx.Argument(2)
	arg3 := ctx.Argument(3)
	arg4 := ctx.Argument(4)

	switch action {
	case "init":
		var check models.User
		err := facades.Orm().Query().FirstOrFail(&check)
		if err == nil {
			color.Redln("面板已初始化")
			return nil
		}

		settings := []models.Setting{{Key: models.SettingKeyName, Value: "耗子Linux面板"}, {Key: models.SettingKeyMonitor, Value: "1"}, {Key: models.SettingKeyMonitorDays, Value: "30"}, {Key: models.SettingKeyBackupPath, Value: "/www/backup"}, {Key: models.SettingKeyWebsitePath, Value: "/www/wwwroot"}, {Key: models.SettingKeyEntrance, Value: "/"}, {Key: models.SettingKeyVersion, Value: facades.Config().GetString("panel.version")}}
		err = facades.Orm().Query().Create(&settings)
		if err != nil {
			color.Redln("初始化失败")
			return nil
		}

		hash, err := facades.Hash().Make(tools.RandomString(32))
		if err != nil {
			color.Redln("初始化失败")
			return nil
		}

		user := services.NewUserImpl()
		_, err = user.Create("admin", hash)
		if err != nil {
			color.Redln("创建管理员失败")
			return nil
		}

		color.Greenln("初始化成功")

	case "update":
		var task models.Task
		err := facades.Orm().Query().Where("status", models.TaskStatusRunning).OrWhere("status", models.TaskStatusWaiting).FirstOrFail(&task)
		if err == nil {
			color.Redln("当前有任务正在执行，禁止更新")
			return nil
		}

		panel, err := tools.GetLatestPanelVersion()
		if err != nil {
			color.Redln("获取最新版本失败")
			return err
		}

		err = tools.UpdatePanel(panel)
		if err != nil {
			color.Redln("更新失败: " + err.Error())
			return nil
		}

		color.Greenln("更新成功")
		tools.RestartPanel()

	case "getInfo":
		var user models.User
		err := facades.Orm().Query().Where("id", 1).FirstOrFail(&user)
		if err != nil {
			color.Redln("获取管理员信息失败")
			return nil
		}

		password := tools.RandomString(16)
		hash, err := facades.Hash().Make(password)
		if err != nil {
			color.Redln("生成密码失败")
			return nil
		}

		user.Username = tools.RandomString(8)
		user.Password = hash

		err = facades.Orm().Query().Save(&user)
		if err != nil {
			color.Redln("保存管理员信息失败")
			return nil
		}

		port := tools.Exec(`cat /www/panel/panel.conf | grep APP_PORT | awk -F '=' '{print $2}' | tr -d '\n'`)

		color.Greenln("用户名: " + user.Username)
		color.Greenln("密码: " + password)
		color.Greenln("面板端口: " + port)
		color.Greenln("面板入口: " + services.NewSettingImpl().Get(models.SettingKeyEntrance, "/"))

	case "getPort":
		port := tools.Exec("cat /www/panel/panel.conf | grep APP_PORT | awk -F '=' '{print $2}'")
		color.Greenln("面板端口: " + port)

	case "getEntrance":
		color.Greenln("面板入口: " + services.NewSettingImpl().Get(models.SettingKeyEntrance, "/"))

	case "deleteEntrance":
		err := services.NewSettingImpl().Set(models.SettingKeyEntrance, "/")
		if err != nil {
			color.Redln("删除面板入口失败")
			return nil
		}

		color.Greenln("删除面板入口成功")

	case "writePlugin":
		slug := arg1
		version := arg2
		if len(slug) == 0 || len(version) == 0 {
			color.Redln("参数错误")
			return nil
		}

		var plugin models.Plugin
		err := facades.Orm().Query().UpdateOrCreate(&plugin, models.Plugin{
			Slug: slug,
		}, models.Plugin{
			Version: version,
		})

		if err != nil {
			color.Redln("写入插件安装状态失败")
			return nil
		}

		color.Greenln("写入插件安装状态成功")

	case "deletePlugin":
		slug := arg1
		if len(slug) == 0 {
			color.Redln("参数错误")
			return nil
		}

		_, err := facades.Orm().Query().Where("slug", slug).Delete(&models.Plugin{})
		if err != nil {
			color.Redln("移除插件安装状态失败")
			return nil
		}

		color.Greenln("移除插件安装状态成功")

	case "writeMysqlPassword":
		password := arg1
		if len(password) == 0 {
			color.Redln("参数错误")
			return nil
		}

		var setting models.Setting
		err := facades.Orm().Query().UpdateOrCreate(&setting, models.Setting{
			Key: models.SettingKeyMysqlRootPassword,
		}, models.Setting{
			Value: password,
		})

		if err != nil {
			color.Redln("写入MySQL root密码失败")
			return nil
		}

		color.Greenln("写入MySQL root密码成功")

	case "cleanTask":
		_, err := facades.Orm().Query().Model(&models.Task{}).Where("status", models.TaskStatusRunning).OrWhere("status", models.TaskStatusWaiting).Update("status", models.TaskStatusFailed)
		if err != nil {
			color.Redln("清理任务失败")
			return nil
		}

		color.Greenln("清理任务成功")

	case "backup":
		backupType := arg1
		name := arg2
		path := arg3
		save := arg4
		hr := `+----------------------------------------------------`
		if len(backupType) == 0 || len(name) == 0 || len(path) == 0 || len(save) == 0 {
			color.Redln("参数错误")
			return nil
		}

		color.Greenln(hr)
		color.Greenln("★ 开始备份 [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

		if !tools.Exists(path) {
			tools.Mkdir(path, 0644)
		}

		switch backupType {
		case "website":
			color.Yellowln("|-目标网站: " + name)
			var website models.Website
			if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err != nil {
				color.Redln("|-网站不存在")
				color.Greenln(hr)
				return nil
			}

			backupFile := path + "/" + website.Name + "_" + carbon.Now().ToShortDateTimeString() + ".zip"
			tools.Exec(`cd '` + website.Path + `' && zip -r '` + backupFile + `' .`)
			color.Greenln("|-备份成功")

		case "mysql":
			rootPassword := services.NewSettingImpl().Get(models.SettingKeyMysqlRootPassword)
			backupFile := name + "_" + carbon.Now().ToShortDateTimeString() + ".sql"

			err := os.Setenv("MYSQL_PWD", rootPassword)
			if err != nil {
				color.Redln("|-备份MySQL数据库失败: " + err.Error())
				color.Greenln(hr)
				return nil
			}

			color.Greenln("|-目标MySQL数据库: " + name)
			color.Greenln("|-开始导出")
			tools.Exec(`mysqldump -uroot ` + name + ` > /tmp/` + backupFile + ` 2>&1`)
			color.Greenln("|-导出成功")
			color.Greenln("|-开始压缩")
			tools.Exec("cd /tmp && zip -r " + backupFile + ".zip " + backupFile)
			tools.Remove("/tmp/" + backupFile)
			color.Greenln("|-压缩成功")
			color.Greenln("|-开始移动")
			if _, err := tools.Mv("/tmp/"+backupFile+".zip", path+"/"+backupFile+".zip"); err != nil {
				color.Redln("|-移动失败: " + err.Error())
				return nil
			}
			color.Greenln("|-移动成功")
			_ = os.Unsetenv("MYSQL_PWD")
			color.Greenln("|-备份成功")

		case "postgresql":
			backupFile := name + "_" + carbon.Now().ToShortDateTimeString() + ".sql"
			check := tools.Exec(`su - postgres -c "psql -l" 2>&1`)
			if strings.Contains(check, name) {
				color.Redln("|-数据库不存在")
				color.Greenln(hr)
				return nil
			}

			color.Greenln("|-目标PostgreSQL数据库: " + name)
			color.Greenln("|-开始导出")
			tools.Exec(`su - postgres -c "pg_dump '` + name + `'" > /tmp/` + backupFile + ` 2>&1`)
			color.Greenln("|-导出成功")
			color.Greenln("|-开始压缩")
			tools.Exec("cd /tmp && zip -r " + backupFile + ".zip " + backupFile)
			tools.Remove("/tmp/" + backupFile)
			color.Greenln("|-压缩成功")
			color.Greenln("|-开始移动")
			if _, err := tools.Mv("/tmp/"+backupFile+".zip", path+"/"+backupFile+".zip"); err != nil {
				color.Redln("|-移动失败: " + err.Error())
				return nil
			}
			color.Greenln("|-移动成功")
			color.Greenln("|-备份成功")
		}

		color.Greenln(hr)
		files, err := os.ReadDir(path)
		if err != nil {
			color.Redln("|-清理失败: " + err.Error())
			return nil
		}
		var filteredFiles []os.FileInfo
		for _, file := range files {
			if strings.HasPrefix(file.Name(), name) && strings.HasSuffix(file.Name(), ".zip") {
				fileInfo, err := os.Stat(filepath.Join(path, file.Name()))
				if err != nil {
					continue
				}
				filteredFiles = append(filteredFiles, fileInfo)
			}
		}
		sort.Slice(filteredFiles, func(i, j int) bool {
			return filteredFiles[i].ModTime().After(filteredFiles[j].ModTime())
		})
		for i := cast.ToInt(save); i < len(filteredFiles); i++ {
			fileToDelete := filepath.Join(path, filteredFiles[i].Name())
			color.Yellowln("|-清理备份: " + fileToDelete)
			tools.Remove(fileToDelete)
		}
		color.Greenln("|-清理完成")
		color.Greenln(hr)
		color.Greenln("☆ 备份完成 [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

	case "cutoff":
		name := arg1
		save := arg2
		hr := `+----------------------------------------------------`
		if len(name) == 0 || len(save) == 0 {
			color.Redln("参数错误")
			return nil
		}

		color.Greenln(hr)
		color.Greenln("★ 开始切割 [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

		color.Yellowln("|-目标网站: " + name)
		var website models.Website
		if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err != nil {
			color.Redln("|-网站不存在")
			color.Greenln(hr)
			return nil
		}

		logPath := "/www/wwwlogs/" + website.Name + ".log"
		if !tools.Exists(logPath) {
			color.Redln("|-日志文件不存在")
			color.Greenln(hr)
			return nil
		}

		backupPath := "/www/wwwlogs/" + website.Name + "_" + carbon.Now().ToShortDateTimeString() + ".log.zip"
		tools.Exec(`cd /www/wwwlogs && zip -r ` + backupPath + ` ` + website.Name + ".log")
		tools.Exec(`echo "" > ` + logPath)
		color.Greenln("|-切割成功")

		color.Greenln(hr)
		files, err := os.ReadDir("/www/wwwlogs")
		if err != nil {
			color.Redln("|-清理失败: " + err.Error())
			return nil
		}
		var filteredFiles []os.FileInfo
		for _, file := range files {
			if strings.HasPrefix(file.Name(), website.Name) && strings.HasSuffix(file.Name(), ".log.zip") {
				fileInfo, err := os.Stat(filepath.Join("/www/wwwlogs", file.Name()))
				if err != nil {
					continue
				}
				filteredFiles = append(filteredFiles, fileInfo)
			}
		}
		sort.Slice(filteredFiles, func(i, j int) bool {
			return filteredFiles[i].ModTime().After(filteredFiles[j].ModTime())
		})
		for i := cast.ToInt(save); i < len(filteredFiles); i++ {
			fileToDelete := filepath.Join("/www/wwwlogs", filteredFiles[i].Name())
			color.Yellowln("|-清理日志: " + fileToDelete)
			tools.Remove(fileToDelete)
		}
		color.Greenln("|-清理完成")
		color.Greenln(hr)
		color.Greenln("☆ 切割完成 [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

	case "writeSite":
		name := arg1
		status := cast.ToBool(arg2)
		path := ctx.Argument(3)
		php := cast.ToInt(ctx.Argument(4))
		ssl := cast.ToBool(ctx.Argument(5))
		if len(name) == 0 || len(path) == 0 {
			color.Redln("参数错误")
			return nil
		}

		var website models.Website
		if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err == nil {
			color.Redln("网站已存在")
			return nil
		}

		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			color.Redln("网站目录不存在")
			return nil
		}

		err = facades.Orm().Query().Create(&models.Website{
			Name:   name,
			Status: status,
			Path:   path,
			Php:    php,
			Ssl:    ssl,
		})
		if err != nil {
			color.Redln("写入网站失败")
			return nil
		}

		color.Greenln("写入网站成功")

	case "deleteSite":
		name := arg1
		if len(name) == 0 {
			color.Redln("参数错误")
			return nil
		}

		_, err := facades.Orm().Query().Where("name", name).Delete(&models.Website{})
		if err != nil {
			color.Redln("删除网站失败")
			return nil
		}

		color.Greenln("删除网站成功")

	case "writeSetting":
		key := arg1
		value := arg2
		if len(key) == 0 || len(value) == 0 {
			color.Redln("参数错误")
			return nil
		}

		var setting models.Setting
		err := facades.Orm().Query().UpdateOrCreate(&setting, models.Setting{
			Key: key,
		}, models.Setting{
			Value: value,
		})
		if err != nil {
			color.Redln("写入设置失败")
			return nil
		}

		color.Greenln("写入设置成功")

	case "getSetting":
		key := arg1
		if len(key) == 0 {
			color.Redln("参数错误")
			return nil
		}

		var setting models.Setting
		if err := facades.Orm().Query().Where("key", key).FirstOrFail(&setting); err != nil {
			return nil
		}

		fmt.Printf("%s", setting.Value)

	case "deleteSetting":
		key := arg1
		if len(key) == 0 {
			color.Redln("参数错误")
			return nil
		}

		_, err := facades.Orm().Query().Where("key", key).Delete(&models.Setting{})
		if err != nil {
			color.Redln("删除设置失败")
			return nil
		}

		color.Greenln("删除设置成功")

	default:
		color.Yellowln(facades.Config().GetString("panel.name") + "命令行工具 - " + facades.Config().GetString("panel.version"))
		color.Greenln("请使用以下命令：")
		color.Greenln("panel update 更新 / 修复面板到最新版本")
		color.Greenln("panel getInfo 重新初始化面板账号信息")
		color.Greenln("panel getPort 获取面板访问端口")
		color.Greenln("panel getEntrance 获取面板访问入口")
		color.Greenln("panel deleteEntrance 删除面板访问入口")
		color.Greenln("panel cleanTask 清理面板运行中和等待中的任务[任务卡住时使用]")
		color.Greenln("panel backup {website/mysql/postgresql} {name} {path} {save_copies} 备份网站 / MySQL数据库 / PostgreSQL数据库到指定目录并保留指定数量")
		color.Greenln("panel cutoff {website_name} {save_copies} 切割网站日志并保留指定数量")
		color.Redln("以下命令请在开发者指导下使用：")
		color.Yellowln("panel init 初始化面板")
		color.Yellowln("panel writePlugin {slug} {version} 写入插件安装状态")
		color.Yellowln("panel deletePlugin {slug} 移除插件安装状态")
		color.Yellowln("panel writeMysqlPassword {password} 写入MySQL root密码")
		color.Yellowln("panel writeSite {name} {status} {path} {php} {ssl} 写入网站数据到面板")
		color.Yellowln("panel deleteSite {name} 删除面板网站数据")
		color.Yellowln("panel getSetting {name} 获取面板设置数据")
		color.Yellowln("panel writeSetting {name} {value} 写入 / 更新面板设置数据")
		color.Yellowln("panel deleteSetting {name} 删除面板设置数据")
	}

	return nil
}

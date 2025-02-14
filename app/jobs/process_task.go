package jobs

import (
	"os/exec"
	"time"

	"github.com/goravel/framework/facades"

	"panel/app/models"
)

// ProcessTask 处理面板任务
type ProcessTask struct {
}

// Signature The name and signature of the job.
func (receiver *ProcessTask) Signature() string {
	return "process_task"
}

// Handle Execute the job.
func (receiver *ProcessTask) Handle(args ...any) error {
	taskID, ok := args[0].(uint)
	if !ok {
		facades.Log().Error("[面板][ProcessTask] 任务ID参数错误")
		return nil
	}

	for {
		if !haveRunningTask() {
			break
		}
		time.Sleep(5 * time.Second)
	}

	var task models.Task
	if err := facades.Orm().Query().Where("id = ?", taskID).Get(&task); err != nil {
		facades.Log().Errorf("[面板][ProcessTask] 获取任务%d失败: %s", taskID, err.Error())
		return nil
	}

	task.Status = models.TaskStatusRunning
	if err := facades.Orm().Query().Save(&task); err != nil {
		facades.Log().Errorf("[面板][ProcessTask] 更新任务%d失败: %s", taskID, err.Error())
		return nil
	}

	facades.Log().Infof("[面板][ProcessTask] 开始执行任务%d", taskID)
	cmd := exec.Command("bash", "-c", task.Shell)
	err := cmd.Run()
	if err != nil {
		task.Status = models.TaskStatusFailed
		if err := facades.Orm().Query().Save(&task); err != nil {
			facades.Log().Errorf("[面板][ProcessTask] 更新任务%d失败: %s", taskID, err.Error())
			return nil
		}
		facades.Log().Errorf("[面板][ProcessTask] 任务%d执行失败: %s", taskID, err.Error())
		return nil
	}

	task.Status = models.TaskStatusSuccess
	if err := facades.Orm().Query().Save(&task); err != nil {
		facades.Log().Errorf("[面板][ProcessTask] 更新任务%d失败: %s", taskID, err.Error())
		return nil
	}

	facades.Log().Infof("[面板][ProcessTask] 任务%d执行成功", taskID)
	return nil
}

// haveRunningTask 是否有任务正在执行
func haveRunningTask() bool {
	var task models.Task
	if err := facades.Orm().Query().Where("status = ?", models.TaskStatusRunning).Get(&task); err != nil {
		facades.Log().Error("[面板][ProcessTask] 获取任务失败: " + err.Error())
		return true
	}

	if task.ID != 0 {
		return true
	}

	return false
}

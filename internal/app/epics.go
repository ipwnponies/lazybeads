package app

import "lazybeads/internal/models"

const epicChildPrefix = "› "

func groupTasksByEpic(tasks []models.Task, allTasks []models.Task) []models.Task {
	if len(tasks) == 0 {
		return tasks
	}

	epicByID := make(map[string]models.Task)
	for _, task := range allTasks {
		if task.Type == "epic" {
			epicByID[task.ID] = task
		}
	}
	if len(epicByID) == 0 {
		return tasks
	}

	epicHasChild := make(map[string]bool)
	for _, task := range tasks {
		if task.EpicParentID != "" {
			epicHasChild[task.EpicParentID] = true
		}
	}

	insertedEpic := make(map[string]bool)
	ordered := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Type == "epic" && epicHasChild[task.ID] {
			continue
		}

		if task.EpicParentID != "" {
			epicID := task.EpicParentID
			if !insertedEpic[epicID] {
				if epic, ok := epicByID[epicID]; ok {
					epic.IsEpicHeader = true
					epic.TreePrefix = ""
					epic.EpicPrefix = ""
					ordered = append(ordered, epic)
				}
				insertedEpic[epicID] = true
			}
			if _, ok := epicByID[epicID]; ok {
				task.EpicPrefix = epicChildPrefix
			}
		}

		if task.Type == "epic" {
			task.IsEpicHeader = true
		}

		ordered = append(ordered, task)
	}

	return ordered
}

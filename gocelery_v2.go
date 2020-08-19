package gocelery

// Delay gets asynchronous result
func (cc *CeleryClient) DelayV2(task string, args ...interface{}) (*AsyncResult, error) {
	celeryTask := getTaskMessageV2(args...)
	headers := getCeleryMessageHeaders(task)
	return cc.delayV2(celeryTask, headers)
}

func (cc *CeleryClient) delayV2(task *TaskMessageV2, headers map[string]interface{}) (*AsyncResult, error) {
	defer releaseTaskMessageV2(task)
	encodedTaskMessage, err := task.Encode()
	if err != nil {
		return nil, err
	}

	celeryMessage := getCeleryMessageV2(encodedTaskMessage, headers)

	defer releaseCeleryMessageV2(celeryMessage)
	err = cc.broker.SendCeleryMessageV2(celeryMessage)
	if err != nil {
		return nil, err
	}
	return &AsyncResult{
		TaskID:  headers["id"].(string),
		backend: cc.backend,
	}, nil
}

package gocelery

// DelayV2 gets asynchronous result
func (cc *CeleryClient) DelayV2(task string, args ...interface{}) (*AsyncResult, error) {
	celeryTask := getTaskMessageV2(args...)
	headers := buildCeleryHeadersV2(task, args, nil)
	return cc.delayV2(celeryTask, headers)
}

// DelayKwargsV2 gets asynchronous result with kwargs support
func (cc *CeleryClient) DelayKwargsV2(task string, kwargs map[string]interface{}, args ...interface{}) (*AsyncResult, error) {
	celeryTask := getTaskMessageV2WithKwargs(args, kwargs)
	headers := buildCeleryHeadersV2(task, args, kwargs)
	return cc.delayV2(celeryTask, headers)
}

func (cc *CeleryClient) delayV2(task *TaskMessageV2, headers *CeleryHeadersV2) (*AsyncResult, error) {
	defer releaseTaskMessageV2(task)
	defer releaseCeleryMessageHeadersV2(headers)

	encodedTaskMessage, err := task.Encode()
	if err != nil {
		return nil, err
	}

	celeryMessage := getCeleryMessageV2(encodedTaskMessage, *headers)

	defer releaseCeleryMessageV2(celeryMessage)
	err = cc.broker.SendCeleryMessageV2(celeryMessage)
	if err != nil {
		return nil, err
	}
	return &AsyncResult{
		TaskID:  headers.ID,
		backend: cc.backend,
	}, nil
}

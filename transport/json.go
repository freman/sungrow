package transport

type jsonMessage struct {
	ResultCode int         `json:"result_code"`
	ResultMsg  string      `json:"result_msg"`
	ResultData interface{} `json:"result_data,omitempty"`
}

type jsonSimpleMessage struct {
	ParamValue string `json:"param_value"`
}

type jsonConnectMessage struct {
	Service     string `json:"service"`
	Token       string `json:"token"`
	UID         int    `json:"uid"`
	TipsDisable int    `json:"tips_disable"`
}

type jsonDeviceListMessage struct {
	Service string `json:"service"`
	List    []struct {
		ID          int           `json:"id"`
		DevID       int           `json:"dev_id"`
		DevCode     int           `json:"dev_code"`
		DevType     int           `json:"dev_type"`
		DevProcotol int           `json:"dev_procotol"`
		InvType     int           `json:"inv_type"`
		DevSn       string        `json:"dev_sn"`
		DevName     string        `json:"dev_name"`
		DevModel    string        `json:"dev_model"`
		PortName    string        `json:"port_name"`
		PhysAddr    string        `json:"phys_addr"`
		LogcAddr    string        `json:"logc_addr"`
		LinkStatus  int           `json:"link_status"`
		InitStatus  int           `json:"init_status"`
		DevSpecial  string        `json:"dev_special"`
		List        []interface{} `json:"list"`
	} `json:"list"`
	Count int `json:"count"`
}

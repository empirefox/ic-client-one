package center

var (
	EMPTY_BYTES = []byte{}
)

var (
	CONNECTING     = []byte(`{"type":"Status","content":"connecting"}`)
	LOGGING_IN     = []byte(`{"type":"Status","content":"logging_in"}`)
	REGGING        = []byte(`{"type":"Status","content":"regging"}`)
	UNREGGING_ROOM = []byte(`{"type":"Status","content":"unregging_room"}`)

	UNREACHABLE    = []byte(`{"type":"Status","content":"unreachable"}`)
	DISCONNECTED   = []byte(`{"type":"Status","content":"disconnected"}`)
	BAD_SERVER_MSG = []byte(`{"type":"Status","content":"bad_server_msg"}`)

	BAD_ROOM_TOKEN        = []byte(`{"type":"Status","content":"bad_room_token"}`)
	REG_ERROR             = []byte(`{"type":"Status","content":"reg_error"}`)
	SAVE_ROOM_TOKEN_ERROR = []byte(`{"type":"Status","content":"save_room_token_error"}`)

	READY = []byte(`{"type":"Status","content":"ready"}`)

	BAD_REG_TOKEN        = []byte(`{"type":"Regable","content":"bad_reg_token"}`)
	SAVE_REG_TOKEN_ERROR = []byte(`{"type":"Regable","content":"save_reg_token_error"}`)
	REGABLE              = []byte(`{"type":"Regable","content":"regable"}`)

	REC_ON  = []byte(`{"type":"RecEnabled","content":1}`)
	REC_OFF = []byte(`{"type":"RecEnabled","content":0}`)
)

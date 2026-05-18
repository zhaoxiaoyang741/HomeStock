package wechat

// BaseInfo is attached to every outgoing iLink API request.
type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

// APIStatus is the common response status from iLink API.
type APIStatus struct {
	Ret     int    `json:"ret,omitempty"`
	Errcode int    `json:"errcode,omitempty"`
	Errmsg  string `json:"errmsg,omitempty"`
}

// Message type constants.
const (
	MessageTypeNone = 0
	MessageTypeUser = 1
	MessageTypeBot  = 2
)

// Message item type constants.
const (
	MessageItemTypeNone  = 0
	MessageItemTypeText  = 1
	MessageItemTypeImage = 2
	MessageItemTypeVoice = 3
	MessageItemTypeFile  = 4
	MessageItemTypeVideo = 5
)

// Message state constants.
const (
	MessageStateNew        = 0
	MessageStateGenerating = 1
	MessageStateFinish     = 2
)

// Upload media type constants.
const (
	UploadMediaTypeImage = 1
	UploadMediaTypeVideo = 2
	UploadMediaTypeFile  = 3
	UploadMediaTypeVoice = 4
)

// TextItem is a text message item.
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// CDNMedia represents a CDN media reference.
type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AesKey            string `json:"aes_key,omitempty"`
	EncryptType       int    `json:"encrypt_type,omitempty"`
	FullURL           string `json:"full_url,omitempty"`
}

// ImageItem is an image message item.
type ImageItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	Aeskey      string    `json:"aeskey,omitempty"`
	Url         string    `json:"url,omitempty"`
	MidSize     int64     `json:"mid_size,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	HDSize      int64     `json:"hd_size,omitempty"`
}

// VoiceItem is a voice message item.
type VoiceItem struct {
	Media         *CDNMedia `json:"media,omitempty"`
	EncodeType    int       `json:"encode_type,omitempty"`
	BitsPerSample int       `json:"bits_per_sample,omitempty"`
	SampleRate    int       `json:"sample_rate,omitempty"`
	Playtime      int       `json:"playtime,omitempty"`
	Text          string    `json:"text,omitempty"`
}

// FileItem is a file message item.
type FileItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	MD5      string    `json:"md5,omitempty"`
	Len      string    `json:"len,omitempty"`
}

// VideoItem is a video message item.
type VideoItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	VideoSize   int64     `json:"video_size,omitempty"`
	PlayLength  int       `json:"play_length,omitempty"`
	VideoMD5    string    `json:"video_md5,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
}

// MessageItem is a single item within a WeixinMessage.
type MessageItem struct {
	Type         int         `json:"type,omitempty"`
	CreateTimeMs int64       `json:"create_time_ms,omitempty"`
	UpdateTimeMs int64       `json:"update_time_ms,omitempty"`
	IsCompleted  bool        `json:"is_completed,omitempty"`
	MsgID        string      `json:"msg_id,omitempty"`
	RefMsg       *RefMessage `json:"ref_msg,omitempty"`
	TextItem     *TextItem   `json:"text_item,omitempty"`
	ImageItem    *ImageItem  `json:"image_item,omitempty"`
	VoiceItem    *VoiceItem  `json:"voice_item,omitempty"`
	FileItem     *FileItem   `json:"file_item,omitempty"`
	VideoItem    *VideoItem  `json:"video_item,omitempty"`
}

// RefMessage is a referenced message.
type RefMessage struct {
	MessageItem *MessageItem `json:"message_item,omitempty"`
	Title       string       `json:"title,omitempty"`
}

// WeixinMessage is a message from the iLink API.
type WeixinMessage struct {
	Seq          int           `json:"seq,omitempty"`
	MessageID    int64         `json:"message_id,omitempty"`
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	CreateTimeMs int64         `json:"create_time_ms,omitempty"`
	UpdateTimeMs int64         `json:"update_time_ms,omitempty"`
	DeleteTimeMs int64         `json:"delete_time_ms,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	GroupID      string        `json:"group_id,omitempty"`
	MessageType  int           `json:"message_type,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
}

// GetUpdatesReq is the request for the long-poll getupdates API.
type GetUpdatesReq struct {
	GetUpdatesBuf string   `json:"get_updates_buf,omitempty"`
	BaseInfo      BaseInfo `json:"base_info,omitempty"`
}

// GetUpdatesResp is the response from the getupdates API.
type GetUpdatesResp struct {
	APIStatus
	Msgs                 []WeixinMessage `json:"msgs,omitempty"`
	GetUpdatesBuf        string          `json:"get_updates_buf,omitempty"`
	LongpollingTimeoutMs int             `json:"longpolling_timeout_ms,omitempty"`
}

// SendMessageReq is the request to send a message.
type SendMessageReq struct {
	Msg      SendMsg  `json:"msg,omitempty"`
	BaseInfo BaseInfo `json:"base_info,omitempty"`
}

// SendMsg is the message payload for sending.
type SendMsg struct {
	ToUserID     string        `json:"to_user_id,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	MessageType  int           `json:"message_type,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
}

// SendMessageResp is the response from sendmessage API.
type SendMessageResp struct {
	APIStatus
}

// GetConfigReq is the request for getconfig API.
type GetConfigReq struct {
	IlinkUserID  string   `json:"ilink_user_id,omitempty"`
	ContextToken string   `json:"context_token,omitempty"`
	BaseInfo     BaseInfo `json:"base_info,omitempty"`
}

// GetConfigResp is the response from getconfig API.
type GetConfigResp struct {
	APIStatus
	TypingTicket string `json:"typing_ticket,omitempty"`
}

// Typing status constants.
const (
	TypingStatusTyping = 1
	TypingStatusCancel = 2
)

// SendTypingReq is the request for sendtyping API.
type SendTypingReq struct {
	IlinkUserID  string   `json:"ilink_user_id,omitempty"`
	TypingTicket string   `json:"typing_ticket,omitempty"`
	Status       int      `json:"status,omitempty"`
	BaseInfo     BaseInfo `json:"base_info,omitempty"`
}

// SendTypingResp is the response from sendtyping API.
type SendTypingResp struct {
	APIStatus
}

// GetUploadUrlReq is the request for getuploadurl API.
type GetUploadUrlReq struct {
	Filekey         string   `json:"filekey,omitempty"`
	MediaType       int      `json:"media_type,omitempty"`
	ToUserID        string   `json:"to_user_id,omitempty"`
	Rawsize         int64    `json:"rawsize,omitempty"`
	RawfileMD5      string   `json:"rawfilemd5,omitempty"`
	Filesize        int64    `json:"filesize,omitempty"`
	ThumbRawsize    int64    `json:"thumb_rawsize,omitempty"`
	ThumbRawfileMD5 string   `json:"thumb_rawfilemd5,omitempty"`
	ThumbFilesize   int64    `json:"thumb_filesize,omitempty"`
	NoNeedThumb     bool     `json:"no_need_thumb,omitempty"`
	Aeskey          string   `json:"aeskey,omitempty"`
	BaseInfo        BaseInfo `json:"base_info,omitempty"`
}

// GetUploadUrlResp is the response from getuploadurl API.
type GetUploadUrlResp struct {
	APIStatus
	UploadParam      string `json:"upload_param,omitempty"`
	ThumbUploadParam string `json:"thumb_upload_param,omitempty"`
	UploadFullURL    string `json:"upload_full_url,omitempty"`
}

// QRCodeResponse is the response from get_bot_qrcode API.
type QRCodeResponse struct {
	Qrcode           string `json:"qrcode"`
	QrcodeImgContent string `json:"qrcode_img_content"`
}

// StatusResponse is the response from get_qrcode_status API.
type StatusResponse struct {
	Status       string `json:"status"` // "wait", "scaned", "confirmed", "expired", "scaned_but_redirect"
	BotToken     string `json:"bot_token,omitempty"`
	IlinkBotID   string `json:"ilink_bot_id,omitempty"`
	Baseurl      string `json:"baseurl,omitempty"`
	IlinkUserID  string `json:"ilink_user_id,omitempty"`
	RedirectHost string `json:"redirect_host,omitempty"`
}

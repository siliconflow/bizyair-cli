package lib

type FileReq struct {
	Sign string `json:"sign,omitempty" form:"sign" query:"sign"`
}

type UserInfo struct {
	Id               string `json:"id" form:"id" query:"id"`
	Name             string `json:"name" form:"name" query:"name"`
	Image            string `json:"image" form:"image" query:"image"`
	Email            string `json:"email" form:"email" query:"email"`
	IsAdmin          bool   `json:"isAdmin" form:"isAdmin" query:"isAdmin"`
	Balance          string `json:"balance" form:"balance" query:"balance"`
	Status           string `json:"status" form:"status" query:"status"`
	Introduction     string `json:"introduction" form:"introduction" query:"introduction"`
	Role             string `json:"role" form:"role" query:"role"`
	ChargeBalance    string `json:"chargeBalance" form:"chargeBalance" query:"chargeBalance"`
	TotalBalance     string `json:"totalBalance" form:"totalBalance" query:"totalBalance"`
	Category         string `json:"category" form:"category" query:"category"`
	CurrentMonthCost string `json:"currentMonthCost" form:"currentMonthCost" query:"currentMonthCost"`
}

type FilesResp struct {
	File    *FileInfo    `json:"file,omitempty" form:"file" query:"file"`
	Storage *StorageInfo `json:"storage,omitempty" form:"storage" query:"storage"`
}

type FileInfo struct {
	Sign            string `json:"sign,omitempty" form:"sign" query:"sign"`
	ObjectKey       string `json:"object_key,omitempty" form:"object_key" query:"object_key"`
	AccessKeyId     string `json:"access_key_id,omitempty" form:"access_key_id" query:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty" form:"access_key_secret" query:"access_key_secret"`
	Expiration      string `json:"expiration,omitempty" form:"expiration" query:"expiration"`
	SecurityToken   string `json:"security_token,omitempty" form:"security_token" query:"security_token"`
	Id              int64  `json:"id,omitempty" form:"id" query:"id"`
}

type StorageInfo struct {
	Endpoint string `json:"endpoint,omitempty" form:"endpoint" query:"endpoint"`
	Bucket   string `json:"bucket,omitempty" form:"bucket" query:"bucket"`
	Region   string `json:"region,omitempty" form:"region" query:"region"`
}

type FileCommitReq struct {
	Sign      string `json:"sign,omitempty" form:"sign" query:"sign"`
	ObjectKey string `json:"object_key,omitempty" form:"object_key" query:"object_key"`
}

type ModelCommitReq struct {
	Name      string       `json:"name,omitempty" form:"name" query:"name"`
	Type      string       `json:"type,omitempty" form:"type" query:"type"`
	Overwrite bool         `json:"overwrite,omitempty" form:"overwrite" query:"overwrite"`
	Files     []*ModelFile `json:"files,omitempty" form:"files" query:"files"`
}

type ModelFile struct {
	Sign string `json:"sign,,omitempty" form:"sign" query:"sign"`
	Path string `json:"path,omitempty" form:"path" query:"path"`
}

type ModelFileInfo struct {
	Path      string `json:"path,omitempty" form:"path" query:"path"`
	LabelPath string `json:"label_path,omitempty" form:"label_path" query:"label_path"`
	RealPath  string `json:"real_path,omitempty" form:"real_path" query:"real_path"`
	Available bool   `json:"available,omitempty" form:"available" query:"available"`
	Sign      string `json:"sign,omitempty" form:"sign" query:"sign"`
}

type ModelCommitResp struct {
}

type ModelDeleteReq struct {
	Name string `json:"name,omitempty" form:"name" query:"name"`
	Type string `json:"type,omitempty" form:"type" query:"type"`
}

type ModelQueryReq struct {
	Name string `json:"name,omitempty" form:"name" query:"name"`
	Type string `json:"type,omitempty" form:"type" query:"type"`
}

type ModelInfo struct {
	Name      string `json:"name,omitempty" form:"name" query:"name"`
	Type      string `json:"type,omitempty" form:"type" query:"type"`
	FileNum   int    `json:"file_num,omitempty" form:"file_num" query:"file_num"`
	Available bool   `json:"available,omitempty" form:"available" query:"available"`
	UpdatedAt string `json:"updated_at,omitempty" form:"updated_at" query:"updated_at"`
}

type ModelListReq struct {
	Type   string `json:"type,omitempty" form:"type" query:"type"`
	Public bool   `json:"public,omitempty" form:"public" query:"public"`
}

type ModelListResp struct {
	Models []*ModelInfo `json:"models,omitempty" form:"models" query:"models"`
}

type ModelListFilesReq struct {
	Type    string `json:"type,omitempty" form:"type" query:"type"`
	Name    string `json:"name,omitempty" path:"name"`
	ExtName string `json:"ext_name,omitempty" path:"ext_name"`
	Public  bool   `json:"public,omitempty" form:"public" query:"public"`
}

type ModelListFilesResp struct {
	Files []*ModelFileInfo `json:"files,omitempty" form:"files" query:"files"`
}

type ModelDeleteResp struct {
}

type CheckModelResp struct {
	Exists bool `json:"exists,omitempty" form:"exists" query:"exists"`
}

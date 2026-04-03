package models

// Single file struct
type FileNode struct {
	Name		string	`json:"name"`					//File name(main.go, data.json etc.)
	Path		string 	`json:"path"`					//File path from the root of the repo
	Type		string	`json:"type"`					//File type
	DownloadUrl	string	`json:"download_url,omitempty"`	//Files raw data url(its empty for directories)
	Size		int64	`json:"size,omitempty"`			//Size of the file
	IsLFS		bool	`json:"is_lfs"`					//is the file contained under the LFS(Large File Storage Protocol)
}
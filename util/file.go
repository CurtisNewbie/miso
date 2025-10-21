package util

import (
	"github.com/curtisnewbie/miso/util/osutil"
)

const (
	// Default File Mode
	DefFileMode = osutil.DefFileMode

	GbUnit uint64 = osutil.GbUnit
	MbUnit uint64 = osutil.MbUnit
	KbUnit uint64 = osutil.KbUnit
)

// Deprecated: since v0.3.6, see osutil pkg.
var (
	FileExists         = osutil.FileExists
	TryFileExists      = osutil.TryFileExists
	ReadFileAll        = osutil.ReadFileAll
	OpenFile           = osutil.OpenFile
	AppendableFile     = osutil.OpenAppendFile
	ReadWriteFile      = osutil.OpenRWFile
	OpenRFile          = osutil.OpenRFile
	OpenRWFile         = osutil.OpenRWFile
	MkdirAll           = osutil.MkdirAll
	MkdirParentAll     = osutil.MkdirParentAll
	SaveTmpFile        = osutil.SaveTmpFile
	FileHasSuffix      = osutil.FileHasSuffix
	FileHasAnySuffix   = osutil.FileHasAnySuffix
	FileAddSuffix      = osutil.FileAddSuffix
	FileReplaceSuffix  = osutil.FileReplaceSuffix
	FileCutSuffix      = osutil.FileCutSuffix
	FileChangeSuffix   = osutil.FileChangeSuffix
	FileCutDotSuffix   = osutil.FileCutDotSuffix
	TempFilePath       = osutil.NewTmpFilePath
	TempFile           = osutil.NewTmpFile
	TempFilePathSuffix = osutil.NewTmpFilePathWith
	TempFileSuffix     = osutil.NewTmpFileWith
	WalkDir            = osutil.WalkDir
	MkdirTree          = osutil.MkdirTree
)

type (
	WalkFsFile = osutil.WalkFsFile
	DirTree    = osutil.DirTree
)

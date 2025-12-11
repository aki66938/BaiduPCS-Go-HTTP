package handler

import (
	"fmt"

	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// matchPath 辅助函数：匹配单条路径
func matchPath(pattern string) (string, error) {
	pcs := pcscommand.GetBaiduPCS()
	user := pcsconfig.Config.ActiveUser()
	paths, err := pcs.MatchPathByShellPattern(user.PathJoin(pattern))
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("path not found")
	}
	return paths[0], nil
}

// matchPaths 辅助函数：匹配多条路径
func matchPaths(patterns ...string) ([]string, error) {
	pcs := pcscommand.GetBaiduPCS()
	user := pcsconfig.Config.ActiveUser()
	var result []string
	for _, p := range patterns {
		paths, err := pcs.MatchPathByShellPattern(user.PathJoin(p))
		if err != nil {
			return nil, err
		}
		result = append(result, paths...)
	}
	return result, nil
}

//go:build windows

package winutil

import (
	"golang.org/x/sys/windows/registry"
)

// QueryAppPathFromRegistry 从注册表 App Paths 查询应用绝对路径
func QueryAppPathFromRegistry(appName string) (string, error) {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\` + appName
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		k, err = registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			return "", err
		}
	}
	defer k.Close()

	val, _, err := k.GetStringValue("")
	if err != nil {
		return "", err
	}
	return val, nil
}

// QueryUninstallDisplayName 从 Uninstall 注册表项根据 DisplayName 查找 InstallLocation
func QueryUninstallDisplayName(targetName string) (string, error) {
	roots := []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE}
	subKeys := []string{
		`Software\Microsoft\Windows\CurrentVersion\Uninstall`,
		`Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, root := range roots {
		for _, sub := range subKeys {
			k, err := registry.OpenKey(root, sub, registry.ENUMERATE_SUB_KEYS)
			if err != nil {
				continue
			}
			names, err := k.ReadSubKeyNames(-1)
			k.Close()
			if err != nil {
				continue
			}

			for _, name := range names {
				appKey, err := registry.OpenKey(root, sub+`\`+name, registry.QUERY_VALUE)
				if err != nil {
					continue
				}
				displayName, _, _ := appKey.GetStringValue("DisplayName")
				installLoc, _, _ := appKey.GetStringValue("InstallLocation")
				displayIcon, _, _ := appKey.GetStringValue("DisplayIcon")
				appKey.Close()

				if displayName == targetName {
					if installLoc != "" {
						return installLoc, nil
					}
					if displayIcon != "" {
						return displayIcon, nil
					}
				}
			}
		}
	}
	return "", nil
}

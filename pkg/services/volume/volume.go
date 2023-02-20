package volume

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"tkestack.io/gpu-manager/pkg/services/volume/ldcache"
	"tkestack.io/gpu-manager/pkg/types"

	"k8s.io/klog"
)

// VolumeManager manages volumes used by containers running GPU application
// 卷管理器管理被运用到GPU应用上的数据卷
type VolumeManager struct {
	Config  []Config `json:"volume,omitempty"`
	cfgPath string

	cudaControlFile string
	cudaSoname      map[string]string
	mlSoName        map[string]string
	share           bool
}

type components map[string][]string

// Config contains volume details in config file
type Config struct {
	Name       string     `json:"name,omitempty"`
	Mode       string     `json:"mode,omitempty"`
	Components components `json:"components,omitempty"`
	BasePath   string     `json:"base,omitempty"`
}

const (
	binDir   = "bin"
	lib32Dir = "lib"
	lib64Dir = "lib64"
)

type volumeDir struct {
	name  string
	files []string
}

// Volume contains directory and file info of volume
type Volume struct {
	Path string
	dirs []volumeDir
}

// VolumeMap stores Volume for each type
type VolumeMap map[string]*Volume

// NewVolumeManager returns a new VolumeManager
// 创建一个新的卷管理器
/*
	一些解释：
	当Pod指定到某个节点上时，首先创建的是一个emptyDir卷，
	并且只要Pod在该节点上运行，卷就一直存在。就像它的名称表示的那样，卷最初是空的。
	尽管Pod中的容器挂载emptyDir卷的路径可能相同也可能不同，但是这些容器都可以
	读写emptyDir卷中相同的文件。当Pod因为某些原因被从节点上删除时，emptyDir卷中的数据也会永久删除
*/

func NewVolumeManager(config string, share bool) (*VolumeManager, error) {
	f, err := os.Open(config) //阅读打开打开指定的文件，如果成功。方法返回的文件可以用于阅读;如果错误产生一个返回值
	if err != nil {
		return nil, err
	}

	defer f.Close() //defer表示最后执行

	volumeManager := &VolumeManager{
		cfgPath:    filepath.Dir(config), //返回卷配置路径的地址
		cudaSoname: make(map[string]string),
		mlSoName:   make(map[string]string),
		share:      share,
	}

	if err := json.NewDecoder(f).Decode(volumeManager); err != nil {
		return nil, err
	}

	return volumeManager, nil
}

// Run starts a VolumeManager
// 开启卷管理器
func (vm *VolumeManager) Run() (err error) {
	cache, err := ldcache.Open()
	if err != nil {
		return err
	}

	defer func() {
		if e := cache.Close(); err == nil {
			err = e
		}
	}()

	vols := make(VolumeMap)
	for _, cfg := range vm.Config {
		vol := &Volume{
			Path: path.Join(cfg.BasePath, cfg.Name),
		}

		if cfg.Name == "nvidia" {
			types.DriverLibraryPath = filepath.Join(cfg.BasePath, cfg.Name)
		} else {
			types.DriverOriginLibraryPath = filepath.Join(cfg.BasePath, cfg.Name)
		}

		for t, c := range cfg.Components {
			switch t {
			case "binaries":
				bins, err := which(c...)
				if err != nil {
					return err
				}

				klog.V(2).Infof("Find binaries: %+v", bins)

				vol.dirs = append(vol.dirs, volumeDir{binDir, bins})
			case "libraries":
				libs32, libs64 := cache.Lookup(c...)
				klog.V(2).Infof("Find 32bit libraries: %+v", libs32)
				klog.V(2).Infof("Find 64bit libraries: %+v", libs64)

				vol.dirs = append(vol.dirs, volumeDir{lib32Dir, libs32}, volumeDir{lib64Dir, libs64})
			}

			vols[cfg.Name] = vol
		}
	}

	if err := vm.mirror(vols); err != nil {
		return err
	}

	klog.V(2).Infof("Volume manager is running")

	return nil
}

// #lizard forgives
func (vm *VolumeManager) mirror(vols VolumeMap) error {
	for driver, vol := range vols {
		if exist, _ := vol.exist(); !exist {
			if err := os.MkdirAll(vol.Path, 0755); err != nil {
				return err
			}
		}

		for _, d := range vol.dirs {
			vpath := path.Join(vol.Path, d.name)
			if err := os.MkdirAll(vpath, 0755); err != nil {
				return err
			}

			// For each file matching the volume components (blacklist excluded), create a hardlink/copy
			// of it inside the volume directory. We also need to create soname symlinks similar to what
			// ldconfig does since our volume will only show up at runtime.
			for _, f := range d.files {
				klog.V(2).Infof("Mirror %s to %s", f, vpath)
				if err := vm.mirrorFiles(driver, vpath, f); err != nil {
					return err
				}

				if strings.HasPrefix(path.Base(f), "libcuda.so") {
					driverStr := strings.SplitN(strings.TrimPrefix(path.Base(f), "libcuda.so."), ".", 2)
					types.DriverVersionMajor, _ = strconv.Atoi(driverStr[0])
					types.DriverVersionMinor, _ = strconv.Atoi(driverStr[1])
					klog.V(2).Infof("Driver version: %d.%d", types.DriverVersionMajor, types.DriverVersionMinor)
				}

				if strings.HasPrefix(path.Base(f), "libcuda-control.so") {
					vm.cudaControlFile = f
				}
			}
		}
	}

	vCudaFileFn := func(soFile string) error {
		if err := os.Remove(soFile); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		if err := clone(vm.cudaControlFile, soFile); err != nil {
			return err
		}

		klog.V(2).Infof("Vcuda %s to %s", vm.cudaControlFile, soFile)

		l := strings.TrimRight(soFile, ".0123456789")
		if err := os.Remove(l); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}
		if err := clone(vm.cudaControlFile, l); err != nil {
			return err
		}
		klog.V(2).Infof("Vcuda %s to %s", vm.cudaControlFile, l)
		return nil
	}

	if vm.share && len(vm.cudaControlFile) > 0 {
		if len(vm.cudaSoname) > 0 {
			for _, f := range vm.cudaSoname {
				if err := vCudaFileFn(f); err != nil {
					return err
				}
			}
		}

		if len(vm.mlSoName) > 0 {
			for _, f := range vm.mlSoName {
				if err := vCudaFileFn(f); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// #lizard forgives
func (vm *VolumeManager) mirrorFiles(driver, vpath string, file string) error {
	obj, err := elf.Open(file)
	if err != nil {
		return fmt.Errorf("%s: %v", file, err)
	}
	defer obj.Close()

	ok, err := blacklisted(file, obj)
	if err != nil {
		return fmt.Errorf("%s: %v", file, err)
	}

	if ok {
		return nil
	}

	l := path.Join(vpath, path.Base(file))
	if err := removeFile(l); err != nil {
		return err
	}

	if err := clone(file, l); err != nil {
		return err
	}

	soname, err := obj.DynString(elf.DT_SONAME)
	if err != nil {
		return fmt.Errorf("%s: %v", file, err)
	}

	if len(soname) > 0 {
		l = path.Join(vpath, soname[0])
		if err := linkIfNotSameName(path.Base(file), l); err != nil && !os.IsExist(err) {
			return err
		}

		// XXX Many applications (wrongly) assume that libcuda.so exists (e.g. with dlopen)
		// Hardcode the libcuda symlink for the time being.
		if strings.Contains(driver, "nvidia") {
			// Remove libcuda symbol link
			if vm.share && driver == "nvidia" && strings.HasPrefix(soname[0], "libcuda.so") {
				os.Remove(l)
				vm.cudaSoname[l] = l
			}

			// Remove libnvidia-ml symbol link
			if vm.share && driver == "nvidia" && strings.HasPrefix(soname[0], "libnvidia-ml.so") {
				os.Remove(l)
				vm.mlSoName[l] = l
			}

			// XXX GLVND requires this symlink for indirect GLX support
			// It won't be needed once we have an indirect GLX vendor neutral library.
			if strings.HasPrefix(soname[0], "libGLX_nvidia") {
				l = strings.Replace(l, "GLX_nvidia", "GLX_indirect", 1)
				if err := linkIfNotSameName(path.Base(file), l); err != nil && !os.IsExist(err) {
					return err
				}
			}
		}
	}

	return nil
}

func (v *Volume) exist() (bool, error) {
	_, err := os.Stat(v.Path)
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}

func (v *Volume) remove() error {
	return os.RemoveAll(v.Path)
}

func removeFile(file string) error {
	if err := os.Remove(file); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func linkIfNotSameName(src, dst string) error {
	if path.Base(src) != path.Base(dst) {
		if err := removeFile(dst); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}

		l := strings.TrimRight(dst, ".0123456789")
		if err := removeFile(l); err != nil {
			if !os.IsExist(err) {
				return err
			}
		}

		if err := os.Symlink(src, l); err != nil && !os.IsExist(err) {
			return err
		}

		if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
			return err
		}
	}

	return nil
}

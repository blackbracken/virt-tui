package tui

import (
	"fmt"
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	libvirt "libvirt.org/libvirt-go"

	"github.com/nyanco01/virt-tui/src/virt"
)


var VirtualMachineStatus    map[string]bool

type CPU struct {
    *tview.Box
    usageGraph      [150][500]string
    usage           [500]float64
    vcpus           uint
}

type Mem struct {
    *tview.Box
    usageGraph      [150][500]string
    usage           [500]float64
    maxMem          uint64
    usedMem         uint64
}

type Disk struct {
    *tview.Box
    infos           []virt.Diskinfo
}

type NIC struct {
    *tview.Box
    bwGraphUp       [150][500]string
    bwGraphDown     [150][500]string
    bwUp            [500]int64
    bwDown          [500]int64
    name            string
}


func NotUpVM(name string) *tview.Box {
    box := tview.NewBox().SetBorder(false)
    box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
        tview.Print(screen, name + " is shutdown", x+1, y + (height / 2), width - 2, tview.AlignCenter, tcell.ColorWhite)

        return x + 1, (y - (height / 2)) + 1, width - 2, height -(y - (height / 2)) + 1 - y
    })

    return box
}

func CreatePages(app *tview.Application) *tview.Pages {
    page := tview.NewPages()
    page.SetBorder(false)

    return page
}

// -------------------------------- Info --------------------------------
func NewVMInfo(dom *libvirt.Domain) *tview.Box {
    box := tview.NewBox().SetBorder(false)
    name, err := dom.GetName()
    if err != nil {
        log.Fatalf("failed to get domain name: %v", err)
    }
    id, err := dom.GetID()
    if err != nil {
        log.Fatalf("failed to get domain id: %v", err)
    }
    uuid, err := dom.GetUUIDString()
    if err != nil {
        log.Fatalf("failed to get domain uuid: %v", err)
    }

    box.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
        tview.Print(screen, fmt.Sprintf("Name : %s", name), x+1, y+1, width, tview.AlignLeft, tcell.ColorWhite)
        tview.Print(screen, fmt.Sprintf("ID   : %d", id), x+1, y+2, width, tview.AlignLeft, tcell.ColorWhite)
        tview.Print(screen, fmt.Sprintf("UUID : %s", uuid), x+1, y+3, width, tview.AlignLeft, tcell.ColorWhite)

        return box.GetInnerRect()
    })

    return box
}

// -------------------------------- CPU --------------------------------
func NewCPU(vcpu uint) *CPU {
    ug := [150][500]string{}
    for i := 0; i < 8; i++ {
        for j := 0; j < 500; j++ {
            ug[i][j] = " "
        }
    }
    u := [500]float64{}
    for i := 0; i < 500; i++ {
        u[i] = 0.0
    }

    return &CPU {
        Box:        tview.NewBox(),
        usageGraph: ug,
        usage:      u,
        vcpus:      vcpu,
    }
}

func (c *CPU) Draw(screen tcell.Screen) {
    c.Box.DrawForSubclass(screen, c)
    x, y, w, h := c.GetInnerRect()

    graphHeight := h - 5
    if graphHeight < 0 {
        graphHeight = 0
    }
    brailleGradient := float64(100) / float64(graphHeight * 4)

    // draw graph
    for i := 0; i < w; i++ {
        usage := c.usage[i]
        for j := 0; j < graphHeight; j++ {
            if (usage - (brailleGradient*4)) > 0 {
                c.usageGraph[j][i] = "⣿"
                usage -= (brailleGradient*4)
            } else {
                a := float64(usage / brailleGradient)
                switch {
                case a < 1.0:
                    c.usageGraph[j][i] = " "
                case 1.0 <= a && a < 2.0:
                    c.usageGraph[j][i] = "⣀"
                case 2.0 <= a && a < 3.0:
                    c.usageGraph[j][i] = "⣤"
                case 3.0 <= a && a < 4.0:
                    c.usageGraph[j][i] = "⣶"
                }
                usage = 0
            }
        }
    }

    graph := []string{}

    for i := 0; i <= graphHeight; i++ {
        tmpLine := ""
        for j := w; j > 0; j-- {
            tmpLine += c.usageGraph[graphHeight - i][j]
        }
        graph = append(graph, tmpLine)
    }

    // draw
    tview.Print(screen, "CPU", x, y-1, w, tview.AlignCenter, tcell.NewRGBColor(0, 255, 127))
    tview.Print(screen, "╔══", x, y-1, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 127))
    tview.Print(screen, "══╗", x, y-1, w, tview.AlignRight, tcell.NewRGBColor(0, 255, 127))

    tview.Print(screen, fmt.Sprintf("Guest VM CPU utilization is %.2f", c.usage[0]), x, y, w, tview.AlignCenter, tcell.ColorForestGreen)
    tview.Print(screen, fmt.Sprintf("%d vCPUs ", c.vcpus), x, y, w, tview.AlignRight, tcell.ColorSpringGreen)

    color := setColorGradation(CPU_COLOR, len(graph))
    for i, line := range graph {
        tview.Print(screen, line, x, y+1+i, w, tview.AlignRight, color[i])
    }

    l := len(graph)
    tview.Print(screen, "╚══", x, y+2+l, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 127))
    tview.Print(screen, "══╝", x, y+2+l, w, tview.AlignRight, tcell.NewRGBColor(0, 255, 127))
}

func (c *CPU)Update(u float64) {
    l := len(c.usage)
    _, _, w, _ := c.GetInnerRect()

    if l < w { w = l }
    for i := w-1; i >= 0; i-- {
        c.usage[i+1] = c.usage[i]
    }

    c.usage[0] = u
}

// -------------------------------- Memory --------------------------------
func NewMemory() *Mem {
    ug := [150][500]string{}
    for i := 0; i < 150; i++ {
        for j := 0; j < 500; j++ {
            ug[i][j] = " "
        }
    }
    u := [500]float64{}
    for i := 0; i < 500; i++ {
        u[i] = 0.0
    }

    return &Mem {
        Box:        tview.NewBox(),
        usageGraph: ug,
        usage:      u,
        maxMem:     0,
        usedMem:    0,
    }
}

func (m *Mem)Draw(screen tcell.Screen) {
    m.Box.DrawForSubclass(screen, m)
    x, y, w, h := m.GetInnerRect()

    graphHeight := h - 5
    if graphHeight < 0 {
        graphHeight = 0
    }
    brailleGradient := float64(100) / float64(graphHeight * 4)

    // draw graph
    for i := 0; i < w; i++ {
        usage := m.usage[i]
        for j := 0; j < graphHeight; j++ {
            if (usage - (brailleGradient*4)) > 0 {
                m.usageGraph[j][i] = "⣿"
                usage -= (brailleGradient*4)
            } else {
                a := int(usage / brailleGradient)
                switch {
                case a == 0:
                    m.usageGraph[j][i] = " "
                case a == 1:
                    m.usageGraph[j][i] = "⣀"
                case a == 2:
                    m.usageGraph[j][i] = "⣤"
                case a == 3:
                    m.usageGraph[j][i] = "⣶"
                }
                usage = 0
            }
        }
    }
    graph := []string{}

    for i := 0; i <= graphHeight; i++ {
        tmpLine := ""
        for j := w; j > 0; j-- {
            tmpLine += m.usageGraph[graphHeight - i][j]
        }
        graph = append(graph, tmpLine)
    }

    memMax := float64(m.maxMem / 1000)
    memUsed := float64(m.usedMem / 1000)

    tview.Print(screen, "Memory", x, y-1, w, tview.AlignCenter, tcell.NewRGBColor(254, 89, 19))
    tview.Print(screen, "╔══", x, y-1, w, tview.AlignLeft, tcell.NewRGBColor(254, 89, 19))
    tview.Print(screen, "══╗", x, y-1, w, tview.AlignRight, tcell.NewRGBColor(254, 89, 19))

    tview.Print(screen, fmt.Sprintf("Max %.3f MiB", memMax), x, y, w, tview.AlignRight, tcell.ColorDarkOrange)
    tview.Print(screen, fmt.Sprintf("Used %.3f MiB",memUsed), x, y+1, w, tview.AlignRight, tcell.ColorOrange)

    color := setColorGradation(MEMORY_COLOR, len(graph))
    for i, line := range graph {
        tview.Print(screen, line, x, y+2+i, w, tview.AlignRight, color[i])
    }

    l := len(graph)
    tview.Print(screen, "╚══", x, y+3+l, w, tview.AlignLeft, tcell.NewRGBColor(254, 89, 19))
    tview.Print(screen, "══╝", x, y+3+l, w, tview.AlignRight, tcell.NewRGBColor(254, 89, 19))

}

func (m *Mem)Update(max, used uint64){
    m.maxMem = max
    m.usedMem = used

    l := len(m.usage)
    _, _, w, _ := m.GetInnerRect()

    if l < w { w = l }
    for i := w-1; i >= 0; i-- {
        m.usage[i+1] = m.usage[i]
    }

    // I can't get memory values for a little while after the VM starts up.
    // So I added it to avoid causing panic.
    if max == 0 {
        m.usage[0] = 0.0
    } else {
        m.usage[0] = float64(used * 100 / max)
    }
}

// -------------------------------- Disk --------------------------------
func NewDisk() *Disk {
    return &Disk {
        Box:        tview.NewBox(),
        infos:      []virt.Diskinfo{},
    }
}

func (d *Disk)AddInfo(info virt.Diskinfo) *Disk {
    d.infos = append(d.infos, info)
    return d
}

func (d *Disk)GetInfoSize() int {
    return len(d.infos)
}

func (d *Disk)Draw(screen tcell.Screen) {
    d.Box.DrawForSubclass(screen, d)
    x, y, w, h := d.GetInnerRect()

    tview.Print(screen, "Disk", x, y, w, tview.AlignCenter, tcell.ColorDarkOrange)
    for i, info := range d.infos {
        if h >= 4 {
            usage := float64(info.Allocation) / float64(info.Capacity)

            // create usage bar
            usageBar := ""
            for i := 0; i < int(usage * float64(w)); i++ {
                usageBar += "■"
            }
            // create bar
            Bar := ""
            for i := 0; i < w; i++{
                Bar += "■"
            }

            tview.Print(screen, fmt.Sprintf("File : %s",info.Name), x, y + (i*4) + 1, w, tview.AlignLeft, tcell.ColorOrange)
            tview.Print(screen, fmt.Sprintf("Volume size : %.2f", float64(info.Capacity / (1024 * 1024 * 1024))), x, y + (i*4) + 1, w, tview.AlignRight, tcell.ColorGhostWhite)
            tview.Print(screen, fmt.Sprintf("Used        : %.2f", float64(info.Allocation / (1024 * 1024 * 1024))), x, y + (i*4) + 2, w, tview.AlignRight, tcell.ColorOrange)
            // draw Bar
            tview.Print(screen, Bar, x, y + (i*4) + 3, w, tview.AlignLeft, tcell.NewRGBColor(80, 80, 80))

            color := setColorGradation(DISK_COLOR, int(usage * float64(w)))
            for j := 0; j< int(usage * float64(w)); j++ {
                tview.Print(screen, "■", x+j, y + (i*4) + 3, w, tview.AlignLeft,color[j])
            }

            //tview.Print(screen, usageBar, x, y + (i*4) + 3, w, tview.AlignLeft, tcell.ColorOrange)
        }
        h -= 4
    }
}

// -------------------------- Network interface card ---------------------------
func NewNIC() *NIC {
    bwU := [150][500]string{}
    for i := 0; i < 150; i++ {
        for j := 0; j < 500; j++ {
            bwU[i][j] = " "
        }
    }
    bwD := [150][500]string{}
    for i := 0; i < 150; i++ {
        for j := 0; j < 500; j++ {
            bwD[i][j] = " "
        }
    }

    bwUp := [500]int64{}
    for i := 0; i < 500; i++ {
        bwUp[i] = 0
    }

    bwDown := [500]int64{}
    for i := 0; i < 500; i++ {
        bwDown[i] = 0
    }

    return &NIC {
        Box:                tview.NewBox(),
        bwGraphUp:          bwU,
        bwGraphDown:        bwD,
        bwUp:               bwUp,
        bwDown:             bwDown,
    }
}

func (n *NIC)Draw(screen tcell.Screen) {
    n.Box.DrawForSubclass(screen, n)
    x, y, w, h := n.GetInnerRect()

    var Uploadjudge int64
    var Downloadjudge int64

    graphHeight := int(h/2) - 2
    if graphHeight < 0 {
        graphHeight = 0
    }
    brailleGradient := float64(100) / float64(graphHeight * 4)

    // Upload Bandwidth
    Uploadjudge = 0
    for i := 0; i < 5; i++ {
        Uploadjudge += n.bwUp[i]
    }
    if (Uploadjudge / 5) > (1000 * 1000) {
        for i := 0; i < w; i++ {
            bandwidth := n.bwUp[i]
            for j := 0; j < graphHeight; j++ {
                if bandwidth > int64(1000 * 1000 * 100 / float64(brailleGradient * 4)) {
                    n.bwGraphUp[j][i] = "⣿"
                    bandwidth -= int64(1000 * 1000 * 100 / float64(brailleGradient * 4))
                } else {
                    a := int(bandwidth / int64(1000 * 1000 * 100 / float64(brailleGradient * 4 * 4)))
                    switch {
                    case a == 0:
                        n.bwGraphUp[j][i] = " "
                    case a == 1:
                        n.bwGraphUp[j][i] = "⣀"
                    case a == 2:
                        n.bwGraphUp[j][i] = "⣤"
                    case a == 3:
                        n.bwGraphUp[j][i] = "⣶"
                    }
                    bandwidth = 0
                }
            }
        }
    } else {
        for i := 0; i < w; i++ {
            bandwidth := n.bwUp[i]
            for j := 0; j < graphHeight; j++ {
                if bandwidth > int64(1000 * 1000 / float64(brailleGradient * 4)) {
                    n.bwGraphUp[j][i] = "⣿"
                    bandwidth -= int64(1000 * 1000 / float64(brailleGradient * 4))
                } else {
                    a := int(bandwidth / int64(1000 * 1000 / float64(brailleGradient * 4 * 4)))
                    switch {
                    case a == 0:
                        n.bwGraphUp[j][i] = " "
                    case a == 1:
                        n.bwGraphUp[j][i] = "⣀"
                    case a == 2:
                        n.bwGraphUp[j][i] = "⣤"
                    case a == 3:
                        n.bwGraphUp[j][i] = "⣶"
                    }
                    bandwidth = 0
                }
            }
        }
    }

    // Download Bandwidth
    Downloadjudge = 0
    for i := 0; i < 5; i++ {
        Downloadjudge += n.bwDown[i]
    }

    if (Downloadjudge / 5) > (1000 * 1000) {
        for i := 0; i < w; i++ {
            bandwidth := n.bwDown[i]
            for j := 0; j < graphHeight; j++ {
                if bandwidth > int64(1000 * 1000 * 100 / float64(brailleGradient * 4)) {
                    n.bwGraphDown[j][i] = "⣿"
                    bandwidth -= int64(1000 * 1000 * 100 / float64(brailleGradient * 4))
                } else {
                    a := int(bandwidth / int64(1000 * 1000 * 100 / float64(brailleGradient * 4 * 4)))
                    switch {
                    case a == 0:
                        n.bwGraphDown[j][i] = " "
                    case a == 1:
                        n.bwGraphDown[j][i] = "⠉"
                    case a == 2:
                        n.bwGraphDown[j][i] = "⠛"
                    case a == 3:
                        n.bwGraphDown[j][i] = "⠿"
                    }
                    bandwidth = 0
                }
            }
        }
    } else {
        for i := 0; i < w; i++ {
            bandwidth := n.bwDown[i]
            for j := 0; j < graphHeight; j++ {
                if bandwidth > int64(1000 * 1000 / float64(brailleGradient * 4)) {
                    n.bwGraphDown[j][i] = "⣿"
                    bandwidth -= int64(1000 * 1000 / float64(brailleGradient * 4))
                } else {
                    a := int(bandwidth / int64(1000 * 1000 / float64(brailleGradient * 4 * 4)))
                    switch {
                    case a == 0:
                        n.bwGraphDown[j][i] = " "
                    case a == 1:
                        n.bwGraphDown[j][i] = "⠉"
                    case a == 2:
                        n.bwGraphDown[j][i] = "⠛"
                    case a == 3:
                        n.bwGraphDown[j][i] = "⠿"
                    }
                    bandwidth = 0
                }
            }
        }
    }

    graphUP := []string{}
    for i := 0; i <= graphHeight; i++ {
        tmpLine := ""
        for j := w; j > 0; j-- {
            tmpLine += n.bwGraphUp[graphHeight - i][j]
        }
        graphUP = append(graphUP, tmpLine)
    }

    graphDOWN := []string{}
    for i := 0; i <= graphHeight; i++ {
        tmpLine := ""
        for j := w; j > 0; j-- {
            tmpLine += n.bwGraphDown[i][j]
        }
        graphDOWN = append(graphDOWN, tmpLine)
    }


    tview.Print(screen, "NIC", x, y-1, w, tview.AlignCenter, tcell.NewRGBColor(20, 161, 156))
    tview.Print(screen, "╔══", x, y-1, w, tview.AlignLeft, tcell.NewRGBColor(20, 161, 156))
    tview.Print(screen, "══╗", x, y-1, w, tview.AlignRight, tcell.NewRGBColor(20, 161, 156))
    tview.Print(screen, fmt.Sprintf("Upload : %.2f KiB", float64(n.bwUp[0] / 1000)), x-30, y, w, tview.AlignRight, tcell.NewRGBColor(31, 247, 255))

    colorUP := setColorGradation(NIC_UP_COLOR, len(graphUP))
    for i, line := range graphUP {
        tview.Print(screen, line, x, y+1+i, w, tview.AlignRight, colorUP[i])
    }
    l := len(graphUP)

    tview.Print(screen, fmt.Sprintf("Download : %.2f KiB", float64(n.bwDown[0] / 1000)), x, y, w, tview.AlignRight, tcell.NewRGBColor(141, 232, 237))
    colorDOWN := setColorGradation(NIC_DOWN_COLOR, len(graphDOWN))
    for i, line := range graphDOWN {
        tview.Print(screen, line, x, y+1+l+i, w, tview.AlignRight, colorDOWN[i])
    }
    l += len(graphDOWN)

    tview.Print(screen, "╚══", x, y+h-1, w, tview.AlignLeft, tcell.NewRGBColor(20, 161, 156))
    tview.Print(screen, "══╝", x, y+h-1, w, tview.AlignRight, tcell.NewRGBColor(20, 161, 156))

    if (Uploadjudge / 5) > (1000 * 1000) {
        tview.Print(screen, "100 MiB", x, y+1, w, tview.AlignLeft, tcell.NewRGBColor(31, 247, 255))
        tview.Print(screen, "  1 MiB", x, y+len(graphUP), w, tview.AlignLeft, tcell.NewRGBColor(20, 161, 156))
    } else {
        tview.Print(screen, "  1 MiB", x, y+1, w, tview.AlignLeft, tcell.NewRGBColor(31, 247, 255))
        tview.Print(screen, "  1 KiB", x, y+len(graphUP), w, tview.AlignLeft, tcell.NewRGBColor(20, 161, 156))
    }
    if (Downloadjudge / 5) > (1000 * 1000) {
        tview.Print(screen, "  1 MiB", x, y+1+len(graphUP), w, tview.AlignLeft, tcell.NewRGBColor(80, 70, 149))
        tview.Print(screen, "100 MiB", x, y+l, w, tview.AlignLeft, tcell.NewRGBColor(141, 232, 237))
    } else {
        tview.Print(screen, "  1 KiB", x, y+1+len(graphUP), w, tview.AlignLeft, tcell.NewRGBColor(80, 70, 149))
        tview.Print(screen, "  1 MiB", x, y+l, w, tview.AlignLeft, tcell.NewRGBColor(141, 232, 237))
    }

}

func (n *NIC)Update(upload, download int64) {
    // Upload
    l := len(n.bwUp)
    _, _, w, _ := n.GetInnerRect()
    if l < w { w = l }
    for i := w-1; i >= 0; i-- {
        n.bwUp[i+1] = n.bwUp[i]
    }
    n.bwUp[0] = upload

    // Download
    l = len(n.bwDown)
    _, _, w, _ = n.GetInnerRect()
    if l < w { w = l }
    for i := w-1; i >= 0; i-- {
        n.bwDown[i+1] = n.bwDown[i]
    }
    n.bwDown[0] = download
}


func NewVMStatus(app * tview.Application, dom *libvirt.Domain, name string) tview.Primitive{
    vmstatus := tview.NewFlex().SetDirection(tview.FlexRow)
    //vmstatus.SetTitle(name)
    //vmstatus.SetBorder(true).SetBorderColor(tcell.NewHexColor(16683008))

    domInfo, err := dom.GetInfo()
    if err != nil {
        log.Fatalf("failed to get domain info: %v", err)
    }
    cpu := NewCPU(domInfo.NrVirtCpu)
    mem := NewMemory()
    disk := NewDisk()
    infos := virt.GetDisks(dom)
    for _, info := range infos {
        disk.AddInfo(info)
    }
    nic := NewNIC()

    vmstatus.AddItem(NewVMInfo(dom), 5, 1, false)
    vmstatus.AddItem(cpu, 0, 1, false)
    vmstatus.AddItem(mem, 0, 1, false)
    vmstatus.AddItem(disk, 2 + (4 * disk.GetInfoSize()), 1, false)
    vmstatus.AddItem(nic, 0, 1, false)

    go func() {
        VMStatusUpdate(app, dom, cpu, mem, nic, name)
    }()

    return vmstatus
}

func VMStatusUpdate(app *tview.Application, d *libvirt.Domain, cpu *CPU, mem *Mem, nic *NIC, name string) {
    sec := time.Second

    oldUsage, _ := virt.GetCPUUsage(d)  // cpu
    oldTX, oldRX := virt.GetNICStatus(d)  // nic

    timeCnt := 0
    for range time.Tick(sec) {
        b, _ := d.IsActive()
        if b && (timeCnt > 3) {
            newUsage, cnt := virt.GetCPUUsage(d)  // cpu
            newTX, newRX := virt.GetNICStatus(d)  // nic

            max, used := virt.GetMemUsed(d)  // memory
            app.QueueUpdateDraw(func() {
                cpu.Update(float64((newUsage - oldUsage) / (uint64(cnt) * 10000000)))  // cpu
                mem.Update(max, used)
                nic.Update(newTX - oldTX, newRX - oldRX)
            })

            oldUsage = newUsage  //cpu
            oldTX = newTX
            oldRX = newRX
        }
        timeCnt++
        if !VirtualMachineStatus[name] {
            break
        }
    }
}


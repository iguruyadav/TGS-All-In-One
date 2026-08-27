export namespace main {
	
	export class AuditPeripherals {
	    Monitors: string[];
	    Keyboards: string[];
	    Mice: string[];
	    Webcams: string[];
	    Audio: string[];
	
	    static createFrom(source: any = {}) {
	        return new AuditPeripherals(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Monitors = source["Monitors"];
	        this.Keyboards = source["Keyboards"];
	        this.Mice = source["Mice"];
	        this.Webcams = source["Webcams"];
	        this.Audio = source["Audio"];
	    }
	}
	export class AuditSecurityInfo {
	    AV: string;
	    Firewall: boolean;
	    IsAdmin: boolean;
	    UAC: string;
	    USBStatus: string;
	    RDPStatus: string;
	    BrowserPolicy: string;
	    ScheduleStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditSecurityInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AV = source["AV"];
	        this.Firewall = source["Firewall"];
	        this.IsAdmin = source["IsAdmin"];
	        this.UAC = source["UAC"];
	        this.USBStatus = source["USBStatus"];
	        this.RDPStatus = source["RDPStatus"];
	        this.BrowserPolicy = source["BrowserPolicy"];
	        this.ScheduleStatus = source["ScheduleStatus"];
	    }
	}
	export class AuditNetworkInfo {
	    Adapter: string;
	    IP: string;
	    MAC: string;
	    Gateway: string;
	    Internet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuditNetworkInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Adapter = source["Adapter"];
	        this.IP = source["IP"];
	        this.MAC = source["MAC"];
	        this.Gateway = source["Gateway"];
	        this.Internet = source["Internet"];
	    }
	}
	export class StorageInfo {
	    Label: string;
	    Type: string;
	    Size: string;
	
	    static createFrom(source: any = {}) {
	        return new StorageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Type = source["Type"];
	        this.Size = source["Size"];
	    }
	}
	export class AuditHardwareInfo {
	    CPU: string;
	    GPU: string;
	    MBBrand: string;
	    MBModel: string;
	    MB: string;
	    RAMTotal: string;
	    RAMSlots: string;
	    RAMDetails: string[];
	    BIOS: string;
	    Storage: StorageInfo[];
	
	    static createFrom(source: any = {}) {
	        return new AuditHardwareInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CPU = source["CPU"];
	        this.GPU = source["GPU"];
	        this.MBBrand = source["MBBrand"];
	        this.MBModel = source["MBModel"];
	        this.MB = source["MB"];
	        this.RAMTotal = source["RAMTotal"];
	        this.RAMSlots = source["RAMSlots"];
	        this.RAMDetails = source["RAMDetails"];
	        this.BIOS = source["BIOS"];
	        this.Storage = this.convertValues(source["Storage"], StorageInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AuditSystemInfo {
	    Name: string;
	    User: string;
	    Domain: string;
	    Manufacturer: string;
	    Model: string;
	    OS: string;
	    Build: string;
	    DeviceType: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditSystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.User = source["User"];
	        this.Domain = source["Domain"];
	        this.Manufacturer = source["Manufacturer"];
	        this.Model = source["Model"];
	        this.OS = source["OS"];
	        this.Build = source["Build"];
	        this.DeviceType = source["DeviceType"];
	    }
	}
	export class AuditData {
	    System: AuditSystemInfo;
	    Hardware: AuditHardwareInfo;
	    Network: AuditNetworkInfo;
	    Security: AuditSecurityInfo;
	    Peripherals: AuditPeripherals;
	
	    static createFrom(source: any = {}) {
	        return new AuditData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.System = this.convertValues(source["System"], AuditSystemInfo);
	        this.Hardware = this.convertValues(source["Hardware"], AuditHardwareInfo);
	        this.Network = this.convertValues(source["Network"], AuditNetworkInfo);
	        this.Security = this.convertValues(source["Security"], AuditSecurityInfo);
	        this.Peripherals = this.convertValues(source["Peripherals"], AuditPeripherals);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class BootItem {
	    App: string;
	    Impact: string;
	    Delay: string;
	
	    static createFrom(source: any = {}) {
	        return new BootItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.App = source["App"];
	        this.Impact = source["Impact"];
	        this.Delay = source["Delay"];
	    }
	}
	export class DashboardStatus {
	    AnimationsEnabled: boolean;
	    TransparencyEnabled: boolean;
	    GameModeEnabled: boolean;
	    HAGSEnabled: boolean;
	    BackgroundAppsEnabled: boolean;
	    PowerPlan: string;
	    PageFileManaged: boolean;
	    PageFileSizeMB: number;
	    SysMainEnabled: boolean;
	    MemCompressionEnabled: boolean;
	    PrefetchEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AnimationsEnabled = source["AnimationsEnabled"];
	        this.TransparencyEnabled = source["TransparencyEnabled"];
	        this.GameModeEnabled = source["GameModeEnabled"];
	        this.HAGSEnabled = source["HAGSEnabled"];
	        this.BackgroundAppsEnabled = source["BackgroundAppsEnabled"];
	        this.PowerPlan = source["PowerPlan"];
	        this.PageFileManaged = source["PageFileManaged"];
	        this.PageFileSizeMB = source["PageFileSizeMB"];
	        this.SysMainEnabled = source["SysMainEnabled"];
	        this.MemCompressionEnabled = source["MemCompressionEnabled"];
	        this.PrefetchEnabled = source["PrefetchEnabled"];
	    }
	}
	export class DebloatConfig {
	    Apps: string[];
	    Tweaks: string[];
	    Options: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new DebloatConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Apps = source["Apps"];
	        this.Tweaks = source["Tweaks"];
	        this.Options = source["Options"];
	    }
	}
	export class DriverInfo {
	    Name: string;
	    Class: string;
	    Version: string;
	    Status: string;
	
	    static createFrom(source: any = {}) {
	        return new DriverInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Class = source["Class"];
	        this.Version = source["Version"];
	        this.Status = source["Status"];
	    }
	}
	export class ForensicsConfig {
	    collectEvtx: boolean;
	    parseRestarts: boolean;
	    hardwareCheck: boolean;
	    dockerHyperV: boolean;
	    generateHTML: boolean;
	    zipOutput: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ForensicsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collectEvtx = source["collectEvtx"];
	        this.parseRestarts = source["parseRestarts"];
	        this.hardwareCheck = source["hardwareCheck"];
	        this.dockerHyperV = source["dockerHyperV"];
	        this.generateHTML = source["generateHTML"];
	        this.zipOutput = source["zipOutput"];
	    }
	}
	export class ForensicsResult {
	    success: boolean;
	    message: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ForensicsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.path = source["path"];
	    }
	}
	export class HealthLog {
	    Time: string;
	    Level: string;
	    Source: string;
	    EventID: number;
	    Message: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = source["Time"];
	        this.Level = source["Level"];
	        this.Source = source["Source"];
	        this.EventID = source["EventID"];
	        this.Message = source["Message"];
	    }
	}
	export class NetworkStatsInfo {
	    In: number;
	    Out: number;
	
	    static createFrom(source: any = {}) {
	        return new NetworkStatsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.In = source["In"];
	        this.Out = source["Out"];
	    }
	}
	export class PerformanceStatus {
	    AnimationsEnabled: boolean;
	    BackgroundAppsEnabled: boolean;
	    PageFileSizeMB: number;
	    PageFileMaxMB: number;
	    PageFileManaged: boolean;
	    GlobalRamOptimizerEnabled: boolean;
	    StartupCleanupEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PerformanceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AnimationsEnabled = source["AnimationsEnabled"];
	        this.BackgroundAppsEnabled = source["BackgroundAppsEnabled"];
	        this.PageFileSizeMB = source["PageFileSizeMB"];
	        this.PageFileMaxMB = source["PageFileMaxMB"];
	        this.PageFileManaged = source["PageFileManaged"];
	        this.GlobalRamOptimizerEnabled = source["GlobalRamOptimizerEnabled"];
	        this.StartupCleanupEnabled = source["StartupCleanupEnabled"];
	    }
	}
	export class ProcessInfo {
	    Name: string;
	    PID: number;
	    Mem: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.PID = source["PID"];
	        this.Mem = source["Mem"];
	    }
	}
	export class RAMStatsInfo {
	    Used: string;
	    Percent: number;
	    Available: string;
	
	    static createFrom(source: any = {}) {
	        return new RAMStatsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Used = source["Used"];
	        this.Percent = source["Percent"];
	        this.Available = source["Available"];
	    }
	}
	export class RCAResult {
	    success: boolean;
	    message: string;
	    cause: string;
	    evidence: string;
	    confidence: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new RCAResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.cause = source["cause"];
	        this.evidence = source["evidence"];
	        this.confidence = source["confidence"];
	        this.details = source["details"];
	    }
	}
	export class SecurityCenterStatus {
	    AV: string;
	    RealtimeEnabled: boolean;
	    FirewallEnabled: boolean;
	    UAC: string;
	    IsAdmin: boolean;
	    TelemetryLevel: number;
	
	    static createFrom(source: any = {}) {
	        return new SecurityCenterStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AV = source["AV"];
	        this.RealtimeEnabled = source["RealtimeEnabled"];
	        this.FirewallEnabled = source["FirewallEnabled"];
	        this.UAC = source["UAC"];
	        this.IsAdmin = source["IsAdmin"];
	        this.TelemetryLevel = source["TelemetryLevel"];
	    }
	}
	export class SmtpConfig {
	    server: string;
	    port: number;
	    username: string;
	    password: string;
	    to: string;
	
	    static createFrom(source: any = {}) {
	        return new SmtpConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.to = source["to"];
	    }
	}
	export class SoftwareItem {
	    ID: string;
	    Name: string;
	    Version?: string;
	    Category?: string;
	    SubCategory?: string;
	    Type: string;
	    Package?: string;
	    Path?: string;
	    Args?: string;
	    Description?: string;
	
	    static createFrom(source: any = {}) {
	        return new SoftwareItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Version = source["Version"];
	        this.Category = source["Category"];
	        this.SubCategory = source["SubCategory"];
	        this.Type = source["Type"];
	        this.Package = source["Package"];
	        this.Path = source["Path"];
	        this.Args = source["Args"];
	        this.Description = source["Description"];
	    }
	}
	export class StartupApp {
	    Name: string;
	    Command: string;
	    Source: string;
	    Enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Command = source["Command"];
	        this.Source = source["Source"];
	        this.Enabled = source["Enabled"];
	    }
	}
	export class VirtualMemInfo {
	    Total: string;
	    Used: string;
	    Available: string;
	    Percent: number;
	
	    static createFrom(source: any = {}) {
	        return new VirtualMemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Total = source["Total"];
	        this.Used = source["Used"];
	        this.Available = source["Available"];
	        this.Percent = source["Percent"];
	    }
	}
	export class TopProcessInfo {
	    Name: string;
	    Mem: string;
	
	    static createFrom(source: any = {}) {
	        return new TopProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Mem = source["Mem"];
	    }
	}
	export class StatsData {
	    CPU: number;
	    RAM: RAMStatsInfo;
	    Disk: number;
	    Network: NetworkStatsInfo;
	    TopProcesses: TopProcessInfo[];
	    VirtualMem: VirtualMemInfo;
	    Uptime: string;
	    LastBoot: string;
	
	    static createFrom(source: any = {}) {
	        return new StatsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CPU = source["CPU"];
	        this.RAM = this.convertValues(source["RAM"], RAMStatsInfo);
	        this.Disk = source["Disk"];
	        this.Network = this.convertValues(source["Network"], NetworkStatsInfo);
	        this.TopProcesses = this.convertValues(source["TopProcesses"], TopProcessInfo);
	        this.VirtualMem = this.convertValues(source["VirtualMem"], VirtualMemInfo);
	        this.Uptime = source["Uptime"];
	        this.LastBoot = source["LastBoot"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SystemHealth {
	    StabilityScore: number;
	    LastFailures: HealthLog[];
	    CriticalLogs: HealthLog[];
	
	    static createFrom(source: any = {}) {
	        return new SystemHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.StabilityScore = source["StabilityScore"];
	        this.LastFailures = this.convertValues(source["LastFailures"], HealthLog);
	        this.CriticalLogs = this.convertValues(source["CriticalLogs"], HealthLog);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}


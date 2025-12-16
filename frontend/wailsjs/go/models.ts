export namespace main {
	
	export class ClientCheckResult {
	    success: boolean;
	    installed: boolean;
	    hasClient: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ClientCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.installed = source["installed"];
	        this.hasClient = source["hasClient"];
	    }
	}
	export class DownloadResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class EmbeddedServer {
	    server_id: number;
	    server_uuid: string;
	    server_name: string;
	    server_address: string;
	    server_port: number;
	    minecraft_version: string;
	    description: string;
	    preview_image_url: string;
	    enabled: number;
	
	    static createFrom(source: any = {}) {
	        return new EmbeddedServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_id = source["server_id"];
	        this.server_uuid = source["server_uuid"];
	        this.server_name = source["server_name"];
	        this.server_address = source["server_address"];
	        this.server_port = source["server_port"];
	        this.minecraft_version = source["minecraft_version"];
	        this.description = source["description"];
	        this.preview_image_url = source["preview_image_url"];
	        this.enabled = source["enabled"];
	    }
	}
	export class InstallResult {
	    Success: boolean;
	    AlreadyInstalled: boolean;
	    Message: string;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Success = source["Success"];
	        this.AlreadyInstalled = source["AlreadyInstalled"];
	        this.Message = source["Message"];
	        this.Error = source["Error"];
	    }
	}
	export class JavaValidationResult {
	    Valid: boolean;
	    Version: string;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new JavaValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Valid = source["Valid"];
	        this.Version = source["Version"];
	        this.Error = source["Error"];
	    }
	}
	export class LaunchMinecraftArgs {
	    JavaPath: string;
	    GameArgs: string[];
	    JVMArgs: string[];
	    WorkingDirectory: string;
	    MinecraftVersion: string;
	    HWID: string;
	    LauncherConfig: Record<string, any>;
	    ServerUuid: string;
	    Username: string;
	
	    static createFrom(source: any = {}) {
	        return new LaunchMinecraftArgs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.JavaPath = source["JavaPath"];
	        this.GameArgs = source["GameArgs"];
	        this.JVMArgs = source["JVMArgs"];
	        this.WorkingDirectory = source["WorkingDirectory"];
	        this.MinecraftVersion = source["MinecraftVersion"];
	        this.HWID = source["HWID"];
	        this.LauncherConfig = source["LauncherConfig"];
	        this.ServerUuid = source["ServerUuid"];
	        this.Username = source["Username"];
	    }
	}
	export class LaunchResult {
	    Success: boolean;
	    Error: string;
	
	    static createFrom(source: any = {}) {
	        return new LaunchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Success = source["Success"];
	        this.Error = source["Error"];
	    }
	}
	export class ModData {
	    id: number;
	    name: string;
	    version?: string;
	    filename: string;
	    size: number;
	    server_id: number;
	
	    static createFrom(source: any = {}) {
	        return new ModData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.filename = source["filename"];
	        this.size = source["size"];
	        this.server_id = source["server_id"];
	    }
	}
	export class LauncherDbConfig {
	    success: boolean;
	    config: Record<string, any>;
	    mods: ModData[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherDbConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.config = source["config"];
	        this.mods = this.convertValues(source["mods"], ModData);
	        this.error = source["error"];
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
	
	export class ModsUpdateResult {
	    success: boolean;
	    updated: boolean;
	    modsUpdated: number;
	    modsDir: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModsUpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.updated = source["updated"];
	        this.modsUpdated = source["modsUpdated"];
	        this.modsDir = source["modsDir"];
	        this.error = source["error"];
	    }
	}
	export class ServerData {
	    id: number;
	    name: string;
	    server_name?: string;
	    server_address?: string;
	    server_port: number;
	    minecraft_version: string;
	    description?: string;
	    preview_image_url?: string;
	    server_uuid?: string;
	    loader_enabled: boolean;
	    loader_type: string;
	    loader_version?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.server_name = source["server_name"];
	        this.server_address = source["server_address"];
	        this.server_port = source["server_port"];
	        this.minecraft_version = source["minecraft_version"];
	        this.description = source["description"];
	        this.preview_image_url = source["preview_image_url"];
	        this.server_uuid = source["server_uuid"];
	        this.loader_enabled = source["loader_enabled"];
	        this.loader_type = source["loader_type"];
	        this.loader_version = source["loader_version"];
	    }
	}
	export class ServerModsResult {
	    success: boolean;
	    mods: any[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerModsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.mods = source["mods"];
	        this.error = source["error"];
	    }
	}
	export class Settings {
	    apiBaseUrl: string;
	    serverUuid?: string;
	    minecraftPath: string;
	    javaPath: string;
	    minMemory: number;
	    maxMemory: number;
	    jvmArgs: string[];
	    windowWidth: number;
	    windowHeight: number;
	    resolution: string;
	    customResolution: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.serverUuid = source["serverUuid"];
	        this.minecraftPath = source["minecraftPath"];
	        this.javaPath = source["javaPath"];
	        this.minMemory = source["minMemory"];
	        this.maxMemory = source["maxMemory"];
	        this.jvmArgs = source["jvmArgs"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.resolution = source["resolution"];
	        this.customResolution = source["customResolution"];
	    }
	}
	export class UninstallResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UninstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}

}


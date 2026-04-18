export namespace launcher {
	
	export class WindowResolution {
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class InstanceConfig {
	    resolution: WindowResolution;
	    java: string;
	    java_args: string;
	    custom_jar: string;
	    min_memory: number;
	    max_memory: number;
	    last_server: string;
	    last_user: string;
	    qmserver_host?: string;
	    qmserver_port?: number;
	    is_using_qmserver_cloud?: boolean;
	    is_premium?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstanceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolution = this.convertValues(source["resolution"], WindowResolution);
	        this.java = source["java"];
	        this.java_args = source["java_args"];
	        this.custom_jar = source["custom_jar"];
	        this.min_memory = source["min_memory"];
	        this.max_memory = source["max_memory"];
	        this.last_server = source["last_server"];
	        this.last_user = source["last_user"];
	        this.qmserver_host = source["qmserver_host"];
	        this.qmserver_port = source["qmserver_port"];
	        this.is_using_qmserver_cloud = source["is_using_qmserver_cloud"];
	        this.is_premium = source["is_premium"];
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
	export class Instance {
	    name: string;
	    uuid: string;
	    game_version: string;
	    mod_loader: string;
	    mod_loader_version?: string;
	    config: InstanceConfig;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.uuid = source["uuid"];
	        this.game_version = source["game_version"];
	        this.mod_loader = source["mod_loader"];
	        this.mod_loader_version = source["mod_loader_version"];
	        this.config = this.convertValues(source["config"], InstanceConfig);
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
	
	export class RemoteInstallMeta {
	    category: string;
	    source: string;
	    projectId: string;
	    slug?: string;
	    title?: string;
	    /** HTTPS thumbnail from CurseForge/Modrinth when installed from catalog */
	    iconUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteInstallMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.source = source["source"];
	        this.projectId = source["projectId"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.iconUrl = source["iconUrl"];
	    }
	}

}

export namespace main {
	
	export class AccountInfo {
	    type: string;
	    username: string;
	    status: string;
	    isDefault: boolean;
	    skinModel: string;
	    skinUuid: string;
	    skinUrl: string;
	    email: string;
	    gameAccountId: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.username = source["username"];
	        this.status = source["status"];
	        this.isDefault = source["isDefault"];
	        this.skinModel = source["skinModel"];
	        this.skinUuid = source["skinUuid"];
	        this.skinUrl = source["skinUrl"];
	        this.email = source["email"];
	        this.gameAccountId = source["gameAccountId"];
	    }
	}
	export class CatalogStoreSettings {
	    curseforge_enabled: boolean;
	    modrinth_enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CatalogStoreSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.curseforge_enabled = source["curseforge_enabled"];
	        this.modrinth_enabled = source["modrinth_enabled"];
	    }
	}
	export class CloudGameAccountInfo {
	    id: number;
	    username: string;
	    uuid: string;
	    serverUuid: string;
	    skinModel: string;
	    skinUrl: string;
	    capeUrl: string;
	    mojangUuid: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudGameAccountInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.uuid = source["uuid"];
	        this.serverUuid = source["serverUuid"];
	        this.skinModel = source["skinModel"];
	        this.skinUrl = source["skinUrl"];
	        this.capeUrl = source["capeUrl"];
	        this.mojangUuid = source["mojangUuid"];
	    }
	}
	export class CurseForgeKeySettings {
	    has_effective_key: boolean;
	    key_saved_in_file: boolean;
	    use_my_key_default: boolean;
	    effective_source: string;
	
	    static createFrom(source: any = {}) {
	        return new CurseForgeKeySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_effective_key = source["has_effective_key"];
	        this.key_saved_in_file = source["key_saved_in_file"];
	        this.use_my_key_default = source["use_my_key_default"];
	        this.effective_source = source["effective_source"];
	    }
	}
	export class InventorySlot {
	    slot: number;
	    item: string;
	    count: number;
	    nbt?: string;
	
	    static createFrom(source: any = {}) {
	        return new InventorySlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.item = source["item"];
	        this.count = source["count"];
	        this.nbt = source["nbt"];
	    }
	}
	export class InventoryEntry {
	    server_id: string;
	    server_name?: string;
	    server_version?: string;
	    player_name: string;
	    timestamp: number;
	    main: InventorySlot[];
	    armor: InventorySlot[];
	    offhand?: InventorySlot;
	
	    static createFrom(source: any = {}) {
	        return new InventoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_id = source["server_id"];
	        this.server_name = source["server_name"];
	        this.server_version = source["server_version"];
	        this.player_name = source["player_name"];
	        this.timestamp = source["timestamp"];
	        this.main = this.convertValues(source["main"], InventorySlot);
	        this.armor = this.convertValues(source["armor"], InventorySlot);
	        this.offhand = this.convertValues(source["offhand"], InventorySlot);
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
	export class GameAccountInventoryResponse {
	    inventories: InventoryEntry[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new GameAccountInventoryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inventories = this.convertValues(source["inventories"], InventoryEntry);
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
	export class InstanceDetails {
	    name: string;
	    uuid: string;
	    gameVersion: string;
	    loader: string;
	    loaderVersion: string;
	    dir: string;
	    mods: string[];
	    shaderpacks: string[];
	    resourcepacks: string[];
	    datapacks: string[];
	    modpacks: string[];
	    schematics: string[];
	    configFiles: string[];
	    kubejsFiles: string[];
	    isUsingQMServerCloud: boolean;
	    isPremium: boolean;
	    lastServer: string;
	    lastUser: string;
	    remoteInstalls: Record<string, launcher.RemoteInstallMeta>;
	    catalogCurseforgeEnabled: boolean;
	    catalogModrinthEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstanceDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.uuid = source["uuid"];
	        this.gameVersion = source["gameVersion"];
	        this.loader = source["loader"];
	        this.loaderVersion = source["loaderVersion"];
	        this.dir = source["dir"];
	        this.mods = source["mods"];
	        this.shaderpacks = source["shaderpacks"];
	        this.resourcepacks = source["resourcepacks"];
	        this.datapacks = source["datapacks"];
	        this.modpacks = source["modpacks"];
	        this.schematics = source["schematics"];
	        this.configFiles = source["configFiles"];
	        this.kubejsFiles = source["kubejsFiles"];
	        this.isUsingQMServerCloud = source["isUsingQMServerCloud"];
	        this.isPremium = source["isPremium"];
	        this.lastServer = source["lastServer"];
	        this.lastUser = source["lastUser"];
	        this.remoteInstalls = this.convertValues(source["remoteInstalls"], launcher.RemoteInstallMeta, true);
	        this.catalogCurseforgeEnabled = source["catalogCurseforgeEnabled"];
	        this.catalogModrinthEnabled = source["catalogModrinthEnabled"];
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
	
	
	export class LauncherAPITargetSettings {
	    use_qmserver_cloud: boolean;
	    custom_api_base: string;
	    effective_api_base: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherAPITargetSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.use_qmserver_cloud = source["use_qmserver_cloud"];
	        this.custom_api_base = source["custom_api_base"];
	        this.effective_api_base = source["effective_api_base"];
	    }
	}
	export class LauncherAboutInfo {
	    version: string;
	    os: string;
	    arch: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherAboutInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	    }
	}
	export class NewsItem {
	    id: number;
	    title: string;
	    content: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new NewsItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	    }
	}
	export class RemoteStoreSearchResponse {
	    hits: meta.RemoteStoreHit[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteStoreSearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], meta.RemoteStoreHit);
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
	export class ResourceStoreLinks {
	    curseforgeUrl: string;
	    modrinthUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new ResourceStoreLinks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.curseforgeUrl = source["curseforgeUrl"];
	        this.modrinthUrl = source["modrinthUrl"];
	    }
	}
	export class ServerInfo {
	    id: string;
	    name: string;
	    address: string;
	    port: number;
	    online: boolean;
	    enabled: boolean;
	    gameServerOnline?: boolean;
	    players: number;
	    maxPlayers: number;
	    version: string;
	    modLoader?: string;
	    modLoaderVersion?: string;
	    isPremium?: boolean;
	    serverID?: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.online = source["online"];
	        this.enabled = source["enabled"];
	        this.gameServerOnline = source["gameServerOnline"];
	        this.players = source["players"];
	        this.maxPlayers = source["maxPlayers"];
	        this.version = source["version"];
	        this.modLoader = source["modLoader"];
	        this.modLoaderVersion = source["modLoaderVersion"];
	        this.isPremium = source["isPremium"];
	        this.serverID = source["serverID"];
	    }
	}

}

export namespace meta {
	
	export class RemoteStoreSide {
	    projectId: string;
	    slug: string;
	    pageUrl: string;
	    downloads: number;
	
	    static createFrom(source: any = {}) {
	        return new RemoteStoreSide(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.slug = source["slug"];
	        this.pageUrl = source["pageUrl"];
	        this.downloads = source["downloads"];
	    }
	}
	export class RemoteStoreHit {
	    source: string;
	    projectId: string;
	    slug: string;
	    title: string;
	    summary: string;
	    iconUrl: string;
	    pageUrl: string;
	    downloads: number;
	    cf?: RemoteStoreSide;
	    mr?: RemoteStoreSide;
	
	    static createFrom(source: any = {}) {
	        return new RemoteStoreHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.projectId = source["projectId"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.iconUrl = source["iconUrl"];
	        this.pageUrl = source["pageUrl"];
	        this.downloads = source["downloads"];
	        this.cf = this.convertValues(source["cf"], RemoteStoreSide);
	        this.mr = this.convertValues(source["mr"], RemoteStoreSide);
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


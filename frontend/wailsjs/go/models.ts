export namespace model {
	
	export class AppVersion {
	    version: string;
	    commit: string;
	    buildDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AppVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.buildDate = source["buildDate"];
	    }
	}
	export class AuthContext {
	    user: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = source["user"];
	        this.password = source["password"];
	    }
	}
	export class BatchSettings {
	    maxConcurrency: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxConcurrency = source["maxConcurrency"];
	    }
	}
	export class Category {
	    id: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	    }
	}
	export class ChangePasswordParams {
	    loginName: string;
	    newPassword: string;
	
	    static createFrom(source: any = {}) {
	        return new ChangePasswordParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.newPassword = source["newPassword"];
	    }
	}
	export class Cluster {
	    id: string;
	    alias: string;
	    host: string;
	    port: number;
	    database: string;
	    category: string;
	    sslmode?: string;
	    connectUser?: string;
	
	    static createFrom(source: any = {}) {
	        return new Cluster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.category = source["category"];
	        this.sslmode = source["sslmode"];
	        this.connectUser = source["connectUser"];
	    }
	}
	export class ClusterInput {
	    alias: string;
	    host: string;
	    port: number;
	    database: string;
	    category: string;
	    sslMode: string;
	    connectUser: string;
	
	    static createFrom(source: any = {}) {
	        return new ClusterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.category = source["category"];
	        this.sslMode = source["sslMode"];
	        this.connectUser = source["connectUser"];
	    }
	}
	export class ClusterResult {
	    clusterId: string;
	    alias: string;
	    host: string;
	    category: string;
	    status: string;
	    message: string;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new ClusterResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusterId = source["clusterId"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.category = source["category"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class DBFunction {
	    call: string;
	    execution?: string;
	    name?: string;
	    params?: string[];
	
	    static createFrom(source: any = {}) {
	        return new DBFunction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.call = source["call"];
	        this.execution = source["execution"];
	        this.name = source["name"];
	        this.params = source["params"];
	    }
	}
	export class DBFunctions {
	    createRole: DBFunction;
	    removeRole: DBFunction;
	    grantParents: DBFunction;
	    revokeParents: DBFunction;
	    changePassword: DBFunction;
	
	    static createFrom(source: any = {}) {
	        return new DBFunctions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.createRole = this.convertValues(source["createRole"], DBFunction);
	        this.removeRole = this.convertValues(source["removeRole"], DBFunction);
	        this.grantParents = this.convertValues(source["grantParents"], DBFunction);
	        this.revokeParents = this.convertValues(source["revokeParents"], DBFunction);
	        this.changePassword = this.convertValues(source["changePassword"], DBFunction);
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
	export class Config {
	    version: number;
	    categories: Category[];
	    clusters: Cluster[];
	    dbFunctions: DBFunctions;
	    batch: BatchSettings;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.categories = this.convertValues(source["categories"], Category);
	        this.clusters = this.convertValues(source["clusters"], Cluster);
	        this.dbFunctions = this.convertValues(source["dbFunctions"], DBFunctions);
	        this.batch = this.convertValues(source["batch"], BatchSettings);
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
	export class CreateRoleParams {
	    loginName: string;
	    fullName: string;
	    email: string;
	    parentRole: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateRoleParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.fullName = source["fullName"];
	        this.email = source["email"];
	        this.parentRole = source["parentRole"];
	    }
	}
	
	
	export class EnvImport {
	    host: string;
	    port: number;
	    database: string;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvImport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.user = source["user"];
	    }
	}
	export class GrantParentsParams {
	    loginName: string;
	    parentRoles: string;
	
	    static createFrom(source: any = {}) {
	        return new GrantParentsParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.parentRoles = source["parentRoles"];
	    }
	}
	export class RemoveRoleParams {
	    loginName: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoveRoleParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	    }
	}
	export class RevokeParentsParams {
	    loginName: string;
	    parentRoles: string;
	
	    static createFrom(source: any = {}) {
	        return new RevokeParentsParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.parentRoles = source["parentRoles"];
	    }
	}
	export class RunRequest {
	    operation: string;
	    categoryIds: string[];
	    clusterIds: string[];
	    auth: AuthContext;
	    createRole?: CreateRoleParams;
	    removeRole?: RemoveRoleParams;
	    grantParents?: GrantParentsParams;
	    revokeParents?: RevokeParentsParams;
	    changePassword?: ChangePasswordParams;
	    confirmProduction: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.categoryIds = source["categoryIds"];
	        this.clusterIds = source["clusterIds"];
	        this.auth = this.convertValues(source["auth"], AuthContext);
	        this.createRole = this.convertValues(source["createRole"], CreateRoleParams);
	        this.removeRole = this.convertValues(source["removeRole"], RemoveRoleParams);
	        this.grantParents = this.convertValues(source["grantParents"], GrantParentsParams);
	        this.revokeParents = this.convertValues(source["revokeParents"], RevokeParentsParams);
	        this.changePassword = this.convertValues(source["changePassword"], ChangePasswordParams);
	        this.confirmProduction = source["confirmProduction"];
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
	export class TestConnectionRequest {
	    clusterId: string;
	    auth: AuthContext;
	
	    static createFrom(source: any = {}) {
	        return new TestConnectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusterId = source["clusterId"];
	        this.auth = this.convertValues(source["auth"], AuthContext);
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


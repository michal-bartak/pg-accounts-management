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
	    color?: string;
	    confirm?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.color = source["color"];
	        this.confirm = source["confirm"];
	    }
	}
	export class CategoryInput {
	    label: string;
	    color: string;
	    confirm: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CategoryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.color = source["color"];
	        this.confirm = source["confirm"];
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
	export class ResetConfigParams {
	    loginName: string;
	    configName: string;
	
	    static createFrom(source: any = {}) {
	        return new ResetConfigParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.configName = source["configName"];
	    }
	}
	export class SetConfigParams {
	    loginName: string;
	    configName: string;
	    configValue: string;
	
	    static createFrom(source: any = {}) {
	        return new SetConfigParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.configName = source["configName"];
	        this.configValue = source["configValue"];
	    }
	}
	export class SetAttributeParams {
	    loginName: string;
	    attribute: string;
	
	    static createFrom(source: any = {}) {
	        return new SetAttributeParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.attribute = source["attribute"];
	    }
	}
	export class SetCommentParams {
	    loginName: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new SetCommentParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.comment = source["comment"];
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
	export class OperationSpec {
	    operation: string;
	    createRole?: CreateRoleParams;
	    removeRole?: RemoveRoleParams;
	    grantParents?: GrantParentsParams;
	    revokeParents?: RevokeParentsParams;
	    changePassword?: ChangePasswordParams;
	    setComment?: SetCommentParams;
	    setAttribute?: SetAttributeParams;
	    setConfig?: SetConfigParams;
	    resetConfig?: ResetConfigParams;
	
	    static createFrom(source: any = {}) {
	        return new OperationSpec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.createRole = this.convertValues(source["createRole"], CreateRoleParams);
	        this.removeRole = this.convertValues(source["removeRole"], RemoveRoleParams);
	        this.grantParents = this.convertValues(source["grantParents"], GrantParentsParams);
	        this.revokeParents = this.convertValues(source["revokeParents"], RevokeParentsParams);
	        this.changePassword = this.convertValues(source["changePassword"], ChangePasswordParams);
	        this.setComment = this.convertValues(source["setComment"], SetCommentParams);
	        this.setAttribute = this.convertValues(source["setAttribute"], SetAttributeParams);
	        this.setConfig = this.convertValues(source["setConfig"], SetConfigParams);
	        this.resetConfig = this.convertValues(source["resetConfig"], ResetConfigParams);
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
	export class ClusterOps {
	    clusterId: string;
	    operations: OperationSpec[];
	
	    static createFrom(source: any = {}) {
	        return new ClusterOps(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusterId = source["clusterId"];
	        this.operations = this.convertValues(source["operations"], OperationSpec);
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
	export class ClusterResult {
	    clusterId: string;
	    alias: string;
	    host: string;
	    category: string;
	    status: string;
	    message: string;
	    durationMs: number;
	    queries?: string[];
	
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
	        this.queries = source["queries"];
	    }
	}
	export class ClusterRoleDetail {
	    clusterId: string;
	    alias: string;
	    host: string;
	    category: string;
	    exists: boolean;
	    comment: string;
	    fullName: string;
	    parents: string[];
	    attributes: Record<string, boolean>;
	    settings: Record<string, string>;
	    error?: string;
	    durationMs: number;
	    queries?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ClusterRoleDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusterId = source["clusterId"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.category = source["category"];
	        this.exists = source["exists"];
	        this.comment = source["comment"];
	        this.fullName = source["fullName"];
	        this.parents = source["parents"];
	        this.attributes = source["attributes"];
	        this.settings = source["settings"];
	        this.error = source["error"];
	        this.durationMs = source["durationMs"];
	        this.queries = source["queries"];
	    }
	}
	export class ClustersConfig {
	    clusters: Cluster[];
	    categories: Category[];
	
	    static createFrom(source: any = {}) {
	        return new ClustersConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusters = this.convertValues(source["clusters"], Cluster);
	        this.categories = this.convertValues(source["categories"], Category);
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
	export class CommentField {
	    key: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new CommentField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	    }
	}
	export class TargetSelection {
	    categoryIds: string[];
	    clusterIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new TargetSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryIds = source["categoryIds"];
	        this.clusterIds = source["clusterIds"];
	    }
	}
	export class UISettings {
	    theme: string;
	    commentDefaultView: string;
	
	    static createFrom(source: any = {}) {
	        return new UISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.commentDefaultView = source["commentDefaultView"];
	    }
	}
	export class DBRead {
	    query: string;
	
	    static createFrom(source: any = {}) {
	        return new DBRead(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	    }
	}
	export class DBReads {
	    searchRoles: DBRead;
	    roleDetail: DBRead;
	    roleParents: DBRead;
	
	    static createFrom(source: any = {}) {
	        return new DBReads(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.searchRoles = this.convertValues(source["searchRoles"], DBRead);
	        this.roleDetail = this.convertValues(source["roleDetail"], DBRead);
	        this.roleParents = this.convertValues(source["roleParents"], DBRead);
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
	    setComment: DBFunction;
	    setAttribute: DBFunction;
	    setConfig: DBFunction;
	    resetConfig: DBFunction;
	
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
	        this.setComment = this.convertValues(source["setComment"], DBFunction);
	        this.setAttribute = this.convertValues(source["setAttribute"], DBFunction);
	        this.setConfig = this.convertValues(source["setConfig"], DBFunction);
	        this.resetConfig = this.convertValues(source["resetConfig"], DBFunction);
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
	    dbReads: DBReads;
	    batch: BatchSettings;
	    ui: UISettings;
	    parentRoles: string[];
	    commentFields: CommentField[];
	    targets: TargetSelection;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.categories = this.convertValues(source["categories"], Category);
	        this.clusters = this.convertValues(source["clusters"], Cluster);
	        this.dbFunctions = this.convertValues(source["dbFunctions"], DBFunctions);
	        this.dbReads = this.convertValues(source["dbReads"], DBReads);
	        this.batch = this.convertValues(source["batch"], BatchSettings);
	        this.ui = this.convertValues(source["ui"], UISettings);
	        this.parentRoles = source["parentRoles"];
	        this.commentFields = this.convertValues(source["commentFields"], CommentField);
	        this.targets = this.convertValues(source["targets"], TargetSelection);
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
	
	
	
	
	
	export class RoleBatchRequest {
	    clusters: ClusterOps[];
	    auth: AuthContext;
	    confirmProduction: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RoleBatchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusters = this.convertValues(source["clusters"], ClusterOps);
	        this.auth = this.convertValues(source["auth"], AuthContext);
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
	export class RoleDetailsRequest {
	    loginName: string;
	    categoryIds: string[];
	    clusterIds: string[];
	    auth: AuthContext;
	
	    static createFrom(source: any = {}) {
	        return new RoleDetailsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loginName = source["loginName"];
	        this.categoryIds = source["categoryIds"];
	        this.clusterIds = source["clusterIds"];
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
	export class RoleMatch {
	    clusterId: string;
	    alias: string;
	    host: string;
	    category: string;
	    loginName: string;
	    comment: string;
	    fullName: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RoleMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clusterId = source["clusterId"];
	        this.alias = source["alias"];
	        this.host = source["host"];
	        this.category = source["category"];
	        this.loginName = source["loginName"];
	        this.comment = source["comment"];
	        this.fullName = source["fullName"];
	        this.error = source["error"];
	    }
	}
	export class RoleSearchRequest {
	    term: string;
	    categoryIds: string[];
	    clusterIds: string[];
	    auth: AuthContext;
	
	    static createFrom(source: any = {}) {
	        return new RoleSearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.term = source["term"];
	        this.categoryIds = source["categoryIds"];
	        this.clusterIds = source["clusterIds"];
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
	export class RunRequest {
	    operation: string;
	    createRole?: CreateRoleParams;
	    removeRole?: RemoveRoleParams;
	    grantParents?: GrantParentsParams;
	    revokeParents?: RevokeParentsParams;
	    changePassword?: ChangePasswordParams;
	    setComment?: SetCommentParams;
	    setAttribute?: SetAttributeParams;
	    setConfig?: SetConfigParams;
	    resetConfig?: ResetConfigParams;
	    categoryIds: string[];
	    clusterIds: string[];
	    auth: AuthContext;
	    confirmProduction: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.createRole = this.convertValues(source["createRole"], CreateRoleParams);
	        this.removeRole = this.convertValues(source["removeRole"], RemoveRoleParams);
	        this.grantParents = this.convertValues(source["grantParents"], GrantParentsParams);
	        this.revokeParents = this.convertValues(source["revokeParents"], RevokeParentsParams);
	        this.changePassword = this.convertValues(source["changePassword"], ChangePasswordParams);
	        this.setComment = this.convertValues(source["setComment"], SetCommentParams);
	        this.setAttribute = this.convertValues(source["setAttribute"], SetAttributeParams);
	        this.setConfig = this.convertValues(source["setConfig"], SetConfigParams);
	        this.resetConfig = this.convertValues(source["resetConfig"], ResetConfigParams);
	        this.categoryIds = source["categoryIds"];
	        this.clusterIds = source["clusterIds"];
	        this.auth = this.convertValues(source["auth"], AuthContext);
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


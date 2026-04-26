export namespace main {
	
	export class ConnectionDTO {
	    key: string;
	    pid: number;
	    appName: string;
	    protocol: string;
	    direction: string;
	    local: string;
	    remote: string;
	    state: string;
	    pingMs: number;
	    loss: number;
	    txRate: string;
	    rxRate: string;
	    age: string;
	    paused: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.pid = source["pid"];
	        this.appName = source["appName"];
	        this.protocol = source["protocol"];
	        this.direction = source["direction"];
	        this.local = source["local"];
	        this.remote = source["remote"];
	        this.state = source["state"];
	        this.pingMs = source["pingMs"];
	        this.loss = source["loss"];
	        this.txRate = source["txRate"];
	        this.rxRate = source["rxRate"];
	        this.age = source["age"];
	        this.paused = source["paused"];
	    }
	}
	export class InitialStateDTO {
	    filter: string;
	    intervalMs: number;
	    pingEnabled: boolean;
	    paused: boolean;
	    ready: boolean;
	    scanning: boolean;
	    lastScan: number;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new InitialStateDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = source["filter"];
	        this.intervalMs = source["intervalMs"];
	        this.pingEnabled = source["pingEnabled"];
	        this.paused = source["paused"];
	        this.ready = source["ready"];
	        this.scanning = source["scanning"];
	        this.lastScan = source["lastScan"];
	        this.lastError = source["lastError"];
	    }
	}
	export class PingSampleDTO {
	    time: number;
	    rttMs: number;
	    loss: number;
	
	    static createFrom(source: any = {}) {
	        return new PingSampleDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.rttMs = source["rttMs"];
	        this.loss = source["loss"];
	    }
	}
	export class StatusDTO {
	    ready: boolean;
	    scanning: boolean;
	    lastScan: number;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.scanning = source["scanning"];
	        this.lastScan = source["lastScan"];
	        this.lastError = source["lastError"];
	    }
	}

}


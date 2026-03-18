export namespace core {
	
	export class WindowState {
	    x: number;
	    y: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class Header {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Header(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class MailConfig {
	    mail_from: string;
	    numbering_mail_from: boolean;
	    rcpt_to: string;
	    numbering_rcpt_to: boolean;
	    subject: string;
	    numbering_subject: boolean;
	    timestamp_subject: boolean;
	    body: string;
	    content_type: string;
	    mail_number: number;
	    thread_number: number;
	    interval_ms: number;
	    use_header_envelope: boolean;
	    update_message_id: boolean;
	    custom_headers: Header[];
	
	    static createFrom(source: any = {}) {
	        return new MailConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mail_from = source["mail_from"];
	        this.numbering_mail_from = source["numbering_mail_from"];
	        this.rcpt_to = source["rcpt_to"];
	        this.numbering_rcpt_to = source["numbering_rcpt_to"];
	        this.subject = source["subject"];
	        this.numbering_subject = source["numbering_subject"];
	        this.timestamp_subject = source["timestamp_subject"];
	        this.body = source["body"];
	        this.content_type = source["content_type"];
	        this.mail_number = source["mail_number"];
	        this.thread_number = source["thread_number"];
	        this.interval_ms = source["interval_ms"];
	        this.use_header_envelope = source["use_header_envelope"];
	        this.update_message_id = source["update_message_id"];
	        this.custom_headers = this.convertValues(source["custom_headers"], Header);
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
	export class ServerConfig {
	    smtp: string;
	    port: number;
	    tls: boolean;
	    ssl: boolean;
	    tls_version: string;
	    skip_verify: boolean;
	    auth: boolean;
	    auth_id: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.smtp = source["smtp"];
	        this.port = source["port"];
	        this.tls = source["tls"];
	        this.ssl = source["ssl"];
	        this.tls_version = source["tls_version"];
	        this.skip_verify = source["skip_verify"];
	        this.auth = source["auth"];
	        this.auth_id = source["auth_id"];
	    }
	}
	export class AppConfig {
	    server: ServerConfig;
	    mail: MailConfig;
	    window: WindowState;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = this.convertValues(source["server"], ServerConfig);
	        this.mail = this.convertValues(source["mail"], MailConfig);
	        this.window = this.convertValues(source["window"], WindowState);
	        this.theme = source["theme"];
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
	export class EMLPreview {
	    subject: string;
	    from: string;
	    to: string;
	    content_type: string;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new EMLPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subject = source["subject"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.content_type = source["content_type"];
	        this.body = source["body"];
	    }
	}
	
	
	
	export class VersionCheckResult {
	    current: string;
	    latest: string;
	    update_avail: boolean;
	    download_url?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.update_avail = source["update_avail"];
	        this.download_url = source["download_url"];
	        this.error = source["error"];
	    }
	}

}


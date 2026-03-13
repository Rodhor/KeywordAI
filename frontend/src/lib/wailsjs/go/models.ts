export namespace jobs {
	
	export class Manager {
	
	
	    static createFrom(source: any = {}) {
	        return new Manager(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace keywords {
	
	export class ClusterParams {
	    evalJobId: string;
	    selectedRows: number[];
	
	    static createFrom(source: any = {}) {
	        return new ClusterParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evalJobId = source["evalJobId"];
	        this.selectedRows = source["selectedRows"];
	    }
	}
	export class EvaluationParams {
	    filename: string;
	    campaignDesc: string;
	    excludeKeywords: string[];
	    customInstructions: string;
	
	    static createFrom(source: any = {}) {
	        return new EvaluationParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.campaignDesc = source["campaignDesc"];
	        this.excludeKeywords = source["excludeKeywords"];
	        this.customInstructions = source["customInstructions"];
	    }
	}
	export class JobRef {
	    jobId: string;
	
	    static createFrom(source: any = {}) {
	        return new JobRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	    }
	}
	export class JobStatus {
	    status: string;
	    error?: string;
	    data?: any;
	
	    static createFrom(source: any = {}) {
	        return new JobStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.error = source["error"];
	        this.data = source["data"];
	    }
	}

}


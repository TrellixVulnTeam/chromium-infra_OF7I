
export async function readProjectConfig(project: string): Promise<ProjectConfig> {
    const r = await fetch(`/api/projects/${encodeURIComponent(project)}/config`);
    return await r.json();
}

export interface ProjectConfig {
    project: string;
    monorail: Monorail;
    paths: string[];
}

export interface Monorail {
    project: string;
    displayPrefix: string;
}
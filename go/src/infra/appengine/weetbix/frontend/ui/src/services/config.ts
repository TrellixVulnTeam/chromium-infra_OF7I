// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

export async function readProjectConfig(project: string): Promise<ProjectConfig> {
    const response = await fetch(`/api/projects/${encodeURIComponent(project)}/config`);
    return await response.json();
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
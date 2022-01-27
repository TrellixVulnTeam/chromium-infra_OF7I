// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {CustomQuestion, IssueCategory, IssueCategoryDetail, IssueWizardPersona} from "./IssueWizardTypes";

// this function is used to get the issue list belong to different persona
// when a user group is selected a list of related issue categories will show up
export function GetCategoriesByPersona (categories: IssueCategory[]): Map<IssueWizardPersona, IssueCategoryDetail[]> {
  const categoriesByPersona = new Map<IssueWizardPersona, IssueCategoryDetail[]>();

  categories.forEach((category) => {
    if (category.enabled) {
      const currentIssuePersona = category.persona;
      const currentCategories = categoriesByPersona.get(currentIssuePersona) ?? [];
      currentCategories.push({
        name: category.name,
        description: category.description,
      });
      categoriesByPersona.set(currentIssuePersona, currentCategories);
    }
  });

  return categoriesByPersona;
}

// this function is used to get the customer questions belong to different issue category
// the customer question page will render base on these data
export function GetQuestionsByCategory(categories: IssueCategory[]): Map<string, CustomQuestion[] | null> {
  const questionsByCategory = new Map<string, CustomQuestion[] | null>();
  categories.forEach((category) => {
    questionsByCategory.set(category.name, category.customQuestions ?? null);
  })
  return questionsByCategory;
}

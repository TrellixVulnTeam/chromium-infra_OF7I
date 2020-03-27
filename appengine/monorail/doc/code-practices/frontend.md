# Monorail Frontend Code Practices

This guide documents the code practices used by Chops Workflow team when writing new client code in the Monorail Issue Tracker.

Through this guide, we use [IETF standards language](https://tools.ietf.org/html/bcp14) to represent the requirements level of different recommendations. For example:



*   **Must** - Code that is written is required to follow this rule.
*   **Should** - We recommend that new code follow this rule where possible.
*   **May** - Following this guideline is optional.

[TOC]


## JavaScript


### Follow the Google JavaScript Style Guide

We enforce the use of the [Google JavaScript style guide](https://google.github.io/styleguide/jsguide.html) through ES Lint, with our rules set by [eslint-config-google](https://github.com/google/eslint-config-google).



*   New JavaScript code **must** follow the Google style guide rules enforced by our ES Lint config.
*   In all other cases, JavaScript **should** adhere to the Google style guide.


#### Exceptions



*   While the Google style guide [recommends using a trailing underscore for private JavaScript field names](https://google.github.io/styleguide/jsguide.html#naming-non-constant-field-names), our code **should** start private field names with an underscore instead to be consistent with the convention adopted by open source libraries we depend on.


### Use modern browser features

We generally aim to write a modern codebase that makes use of recent JavaScript features to keep our coding conventions fresh.



*   When using features that are not yet supported in [all supported browsers](#heading=h.s0dpmzuabf7w), we **must** polyfill features to meet our browser support requirements.
*   New JavaScript code **should not** inject values into the global scope. ES modules should be used for importing variables and functions across files instead.
*   When writing asynchronous code, JavaScript code **should** favor async/await over Promises.
    *   Exception: Promise.all() **may** be used to simultaneously run multiple await calls.
*   JavaScript code **should** use the modularized forms of built-in functions rather than the global forms. For example, prefer [Number.parseInt](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/parseInt) over [parseInt](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/parseInt).
*   String building code **should** prefer [ES template literals](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals) over string concatenation for strings built from multiple interpolated variables. Template literals usually produce more readable code than direct concatenation.
*   JavaScript code **should** prefer using native browser functionality over importing external dependencies. If a native browser function does the same thing as an external library, prefer using the native functionality.


#### Browser support guidelines

All code written **must** target the two latest stable versions of each of the following browsers, per the [Chrome Ops Browser Support Guidelines](https://chromium.googlesource.com/infra/infra/+/master/doc/front_end.md#browser-support).


### Avoid unexpected Object mutations



*   Functions **must not** directly mutate any Object or Arrays that are passed into them as parameters, unless the function explicitly documents that it mutates its parameters.
*   Objects and Arrays declared as constants **should **have their values frozen with Object.freeze() to prevent accidental mutations.


### Create readable function names



*   Event handlers **should** be named after the actions they do, not the actions that trigger them. For example starIssue() is a better function header than onClick().
    *   Exception: APIs that allow specifying custom handlers **may** accept event handling functions with generic names like clickHandler.


### Code comments



*   TODOs in code **should** reference a tracking issue in the TODO comment.


### Performance



*   JavaScript code **should not** create extraneous objects in loops or repeated functions. Object initialization is expensive, including for common native objects such as Function, RegExp, and Date. Thus, creating new Objects in every iteration of a loop or other repeated function can lead to unexpected performance hits. Where possible, initialize objects once and re-use them on each iteration rather than recreating them repeatedly.


## Web components/LitElement

Monorail’s frontend is written as a single page app (SPA) using the JavaScript framework [LitElement](https://lit-element.polymer-project.org/). LitElement is a lightweight framework built on top of the native Web Components API present in modern browsers.

When creating new elements, we try to follow recommended code practices for both Web Components and LitElement.


### Web components practices

Google Web Fundamentals offers a [Custom Elements Best Practices](https://developers.google.com/web/fundamentals/web-components/best-practices) guide. While the recommendations in this guide are solid, many of them are already covered by LitElement’s underlying usage of Web Components. Thus, we avoid explicitly requiring following this guide to avoid confusion.

However, many of the recommendations from this guide are useful, even when using LitElement. We adapt this advice for our purposes here.



*   Elements **should not** break the hidden attribute when styling the :host element. For example, if you set a custom element to have the style of `:host { display: flex; }`, by default this causes the `:host[hidden]` state to also use `display: flex;`. To overcome this, you can use CSS to explicitly declare that the :host element should be hidden when the hidden attribute is set.
*   Elements **should** enable Shadow DOM to encapsulate styles. LitElement enables Shadow DOM by default and offers options for disabling it. However, disabling ShadowDOM is discouraged because many Web Components features, such as <slot> elements, are built on top of Shadow DOM.
*   Elements **should not** error when initialized without attributes. For example, adding attribute-less HTML such as `<my-custom-element></my-custom-element>` to the DOM should not cause code errors.
*   Elements **should not** dispatch events in response to changes made by the parent. The parent already knows about its own activity, so dispatching events in these circumstances is not meaningful.
*   Elements **should not** accept rich data (ie: Objects and Arrays) as attributes. LitElement provides APIs for setting Arrays and Objects through either attributes or properties, but it is more efficient to set them through properties. Setting these values through attributes requires extra serialization and deserialization.
    *   Exception: When LitElements are used outside of lit-html rendering, declared HTML **may** pass rich data in through attributes. Outside of lit-html, setting property values for DOM is often inconvenient.
*   Elements **should not** self apply classes to their own :host element. The parent of an element is responsible for applying classes to an element, not the element itself.


### Organizing elements

When creating a single page app using LitElement, we render all components in the app through a root component that handles frontend routing and loads additional components for routes the user visits.



*   Elements **should** be grouped into folders with the same name as the element with tests and element code living together.
    *   Exception: Related sub-elements of a parent element **may** be grouped into the parent element’s folder if the element is not used outside the parent.
*   Pages **should** lazily load dependent modules when the user first navigates to that page. To improve initial load time performance, we split element code into bundles divided across routes.
*   Elements **should** use the mr- prefix if they are specific to Monorail and **may** use the chops- prefix if they implement functionality that is general enough to be shared with other apps. (ie: chops-button, chops-checkbox)
*   Nested routes **may** be subdivided into separate sub-components with their own routing logic. This pattern promotes sharing code between related pages.


### LitElement lifecycle

LitElement provides several lifecycle callbacks for elements. When writing code, it is important to be aware of which kinds of work should be done during different phases of an element’s life.



*   Elements **must** remove any created external side effects before disconnectedCallback() finishes running.
    *   Example: If an element attaches any global event handlers at any time in its lifecycle, those global event handlers **must not** continue to run when the element is removed from the DOM.
*   Elements **should not** do work dependent on property values in connectedCallback() or constructor(). These functions will not re-run if property values change, so work done based on property values will become stale when properties change.
    *   Exception: An element **may** initialize values based on other property values as long as values continue to be updated beyond when the element initializes.
    *   Use update() for functionality meant to run before the render() cycle and updated() for functionality that runs after.
*   Elements **should** use the update() callback for work that happens _before render_ and the updated() callback for work that happens _after render_.
    *   More strictly, code with significant side effects such as XHR requests **must not** run in the update() callback but **may** run in the updated() callback.


### Sharing functionality



*   Elements **should not** use mixins for sharing behavior. See: [Mixins Considered Harmful](https://reactjs.org/blog/2016/07/13/mixins-considered-harmful.html)
    *   Exception: Elements **may** use the connect() mixin for sharing Redux functionality.


### HTML/CSS



*   HTML and CSS written in elements **should** follow the [Google HTML/CSS style guide](https://google.github.io/styleguide/htmlcssguide.html).
*   An element **should** make its :host element the main container of the element. When a parent element styles a child element directly through CSS, the :host element is the HTMLElement that receives the styles.
*   Styles in CSS **should** aim to use the minimum specificity required to apply the declared style. This is important because increased specificity is used to overwrite existing styles. In particular:
    *   CSS **should not** use the !important directive.
    *   Elements **should** be styled with classes rather than with IDs because styles applied to ID selectors are harder to overwrite.
*   CSS custom properties **should** be used to specify commonly used CSS values, such as shared color palette colors.
    *   In addition, CSS custom properties **may** be used by individual elements to expose an API for parents to style the element through.
*   Elements **may** use shared JavaScript constants with lit-html CSS variables to share styles among multiple elements. See: [Sharing styles in LitElement](https://lit-element.polymer-project.org/guide/styles#sharing-styles)


### Security recommendations



*   Code **must not** use LitElement’s unsafeHTML or unsafeCSS directives.
*   Code **must not** directly set anchor href values outside of LitElement’s data binding system. LitElement sanitizes variable values when data binding and manually binding data to the DOM outside of LitElement’s sanitization system is a security liability.
*   Code **must not** directly set innerHTML on any elements to values including user-inputted data.
    *   Note: It is common for [Web Component example code](https://developers.google.com/web/fundamentals/web-components/customelements) to make use of directly setting innerHTML to set HTML templates. In these examples, setting innerHTML is often safe because the sample code does not add any variables into the rendered HTML. However, setting innerHTML directly is still risky and can be completely avoided as a pattern when writing LitElement elements.


## Redux/Data Binding

We use [Redux](https://redux.js.org/) on our LitElement frontend to manage state.



*   JavaScript code **must** maintain unidirectional data flow. Unidirectional data flow could also be referred to as “props down, events up” and other names. See: [Redux Data Flow](https://redux.js.org/basics/data-flow/)
    *   In short, all data that lives in Redux **must** be edited by editing the Redux store, not through intermediate data changes. These edits happen through dispatched actions.
    *   This means that automatic 2-way data binding patterns, used in frameworks like Polymer, **must not** be used.
    *   Note: For component data stored outside of Redux, this data flow pattern **should** still be followed by treating the topmost component where data originated from as the “parent” of the data.
*   JavaScript code **must** follow all rules listed as “Essential” in the [Redux style guide](https://redux.js.org/style-guide/style-guide/).
    *   Additionally, “Strongly Recommended” and “Recommended” rules in Redux’s style guide **may** be followed.
*   Objects that cannot be directly serialized into JSON **must not** be stored in the Redux store. As an example, this includes JavaScript’s native Map and Set object.
*   Reducers, actions, and selectors **must** be organized into files according to the [“Ducks” pattern](https://github.com/erikras/ducks-modular-redux).
*   Reducers **should** be separated into small functions that individually handle only a few action types.
*   Redux state **should** be normalized to avoid storing duplicate copies of the same data in the state. See: [Normalizing Redux State Shape](https://redux.js.org/recipes/structuring-reducers/normalizing-state-shape/)
*   JavaScript code **should not **directly pull data from the Redux state Object. Data inside the Redux store **should** be accessed through a layer of selector functions, using [Reselect](https://github.com/reduxjs/reselect).
*   Reducers, selectors, and action creators **should** be unit tested like functions. For example, a reducer is a function that takes in an initial state and an action then returns a new state.
*   Reducers **may** destructure action arguments to make it easier to read which kinds of action attributes are used. In particular, this pattern is useful when reducers are composed into many small functions that each handle a small number of actions.
*   Components connected to Redux **may** use the [Presentational/Container component pattern](https://medium.com/@dan_abramov/smart-and-dumb-components-7ca2f9a7c7d0) to separate “connected” versions of the component from “unconnected” versions. This pattern is useful for testing components outside of Redux and for separating concerns.


## Testing



*   Mock timers **must** be added for code which depends on time. For example, any code which uses debouncing, throttling, settimeout, or setInterval **must** mock time in tests. Tests dependent on time are likely to flakily fail.
*   New JavaScript code **must** have 90% or higher test coverage. Where possible, code **should** aim for 100% test coverage.
*   Unit tests **should** be kept small when possible. More smaller tests are preferable to fewer larger tests.
*   The HTMLElement dispatchEvent() function **may** be used to simulate browser events such as keypresses, mouse events, and more.


## UX


### Follow Material Design guidelines

When making design changes to our UI, we aim to follow [Google’s Material Design guidelines](https://material.io/design/). In particular, because we are designing a developer tool where our users benefit from power user features and high scannability for large amounts of data, we pay particular attention to [recommendations on applying density](https://material.io/design/layout/applying-density.html).



*   Our UI designs **must not** directly use Google branding such as the Google logo or Google brand colors. Monorail is not an official Google product.
*   Visual designs **should** follow the Material Design guidelines. In particular, try
*   Colors used in designs **should** be taken from the [2014 Material Design color palette](https://material.io/design/color/the-color-system.html). Where this color palette falls short of our design needs, new colors **should** be created by mixing shades of the existing 2014 Material colors.
*   Our UI designs **should** follow a “build from white” design philosophy where most of the UI is neutral in color and additional hues are added to draw emphasis to specific elements.


### Accessibility

To keep our UI accessible to a variety of users, we aim to follow the [WAI-ARIA guidelines](https://www.w3.org/WAI/standards-guidelines/aria/).



*   UI designs **must **keep a 4.5:1 minimum contrast ratio for text and icons.
*   CSS **must not **set “outline: none” without creating a new focus style for intractable elements. While removing native focus styles is a tempting way to make a design “feel more modern”, being able to see which elements are focused is an essential feature for keyboard interaction.
*   UI changes **should** follow the [Material Design accessibility guidelines](https://material.io/design/usability/accessibility.html).
*   HTML code **should** favor using existing semantic native elements over recreating native functionality where possible. For example, it is better to use a <button> element for a clickable button than to create a <div> with an onclick handler.
    *   Tip: In many cases, an underlying native element **may** be used as part of an implementation that otherwise seems to need completely custom code. One common example is when styling native checkboxes: while many CSS examples create new DOM elements to replace the look of a native checkbox, it is possible to use CSS pseudoelements to tie interaction with those new elements to an underlying native checkbox.
    *   Exception: Oftentimes, specific code requirements will make using native elements unfeasible. In these cases, the custom element implementation **must** follow any relevant WAI-ARIA for the type of element being implemented. For example, these are the [WAI-ARIA guidelines on implementing an accessible modal dialog](https://www.w3.org/TR/wai-aria-practices/examples/dialog-modal/dialog.html).
*   Any element with an onclick handler **should** be a <button>. The <button> element handles mapping keyboard shortcuts to click handlers, which adds keyboard support for these intractable elements.
*   Manual screenreader testing **should** be done when implementing heavily customized interactive elements. Autocomplete, chip inputs, and custom modal implementations are examples of elements that should be verified against screenreader testing.


### Writing



*   Error messages **should** guide users towards solving the cause of the error when possible.
*   Text in the UI **should not** use terminology that’s heavily tied to app internals. Concepts **should** be expressed in terms that users understand.
*   Wording **should** be kept simple and short where possible.
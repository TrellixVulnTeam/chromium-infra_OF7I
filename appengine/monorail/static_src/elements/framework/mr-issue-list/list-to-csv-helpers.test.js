// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

import {assert} from 'chai';
import sinon from 'sinon';
import {
  constructHref,
  convertListContentToCsv,
  prepareDataForDownload,
  preventCSVInjectionAndStringify,
  SNIFF_PREVENTION,
} from './list-to-csv-helpers.js';

describe('constructHref', () => {
  it('has default of empty string', () => {
    const result = constructHref();
    assert.equal(result, 'data:text/csv;charset=utf-8,');
  });

  it('starts with data:', () => {
    const result = constructHref('');
    assert.isTrue(result.startsWith('data:'));
  });

  it('uses charset=utf-8', () => {
    const result = constructHref('');
    assert.isTrue(result.search('charset=utf-8') > -1);
  });

  it('encodes URI component', () => {
    const encodeFuncStub = sinon.stub(window, 'encodeURIComponent');
    constructHref('');
    sinon.assert.calledOnce(encodeFuncStub);

    window.encodeURIComponent.restore();
  });

  it('encodes URI component', () => {
    const input = 'foo, bar fizz=buzz';
    const expected = 'foo%2C%20bar%20fizz%3Dbuzz';
    const output = constructHref(input);

    assert.equal(expected, output.split(',')[1]);
  });
});

describe('convertListContentToCsv', () => {
  it('joins rows with new line characters', () => {
    const input = [['foobar'], ['fizzbuzz']];
    const expected = '\n"foobar"\n"fizzbuzz"';
    assert.equal(expected, convertListContentToCsv(input));
  });

  it('joins columns with commas', () => {
    const input = [['foo', 'bar', 'fizz', 'buzz']];
    const expected = '\n"foo","bar","fizz","buzz"';
    assert.equal(expected, convertListContentToCsv(input));
  });

  it('starts with a new line character', () => {
    const input = [['foobar']];
    const expected = '\n"foobar"';
    assert.equal(expected, convertListContentToCsv(input));
  });
});

describe('prepareDataForDownload', () => {
  it('prepends sniff prevention', () => {
    const result = prepareDataForDownload([['a']]);
    assert.equal(SNIFF_PREVENTION, result.split('\n')[0]);
  });

  it('prepends prefix', () => {
    const prefix = 'foobar';
    const result = prepareDataForDownload([['a']], undefined, prefix);
    assert.equal(prefix, result.split('\n')[1]);
  });

  it('prepends header row', () => {
    const headers = ['column1', 'column2'];
    const result = prepareDataForDownload([['a', 'b']], headers);

    const expected = `"column1","column2"`;
    assert.equal(expected, result.split('\n')[1]);
  });
});

describe('preventCSVInjectionAndStringify', () => {
  it('prepends all double quotes with another double quote', () => {
    let input = '"hello world"';
    let expect = '""hello world""';
    assert.equal(expect, preventCSVInjectionAndStringify(input).slice(1, -1));

    input = 'Just a double quote: " ';
    expect = 'Just a double quote: "" ';
    assert.equal(expect, preventCSVInjectionAndStringify(input).slice(1, -1));

    input = 'Multiple"double"quotes"""';
    expect = 'Multiple""double""quotes""""""';
    assert.equal(expect, preventCSVInjectionAndStringify(input).slice(1, -1));
  });

  it('wraps string with double quotes', () => {
    let input = '"hello world"';
    let expected = preventCSVInjectionAndStringify(input);
    assert.equal('"', expected[0]);
    assert.equal('"', expected[expected.length-1]);

    input = 'For unevent quotes too: " ';
    expected = '"For unevent quotes too: "" "';
    assert.equal(expected, preventCSVInjectionAndStringify(input));

    input = 'And for ending quotes"""';
    expected = '"And for ending quotes"""""""';
    assert.equal(expected, preventCSVInjectionAndStringify(input));
  });

  it('wraps strings containing commas with double quotes', () => {
    const input = 'Let\'s, add, a bunch, of, commas,';
    const expected = '"Let\'s, add, a bunch, of, commas,"';
    assert.equal(expected, preventCSVInjectionAndStringify(input));
  });

  it('can handle strings containing commas and new line chars', () => {
    const input = `""new"",\nline  "" "",\nand 'end', and end`;
    const expected = `"""""new"""",\nline  """" """",\nand 'end', and end"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));
  });

  it('preserves single quotes', () => {
    let input = `all the 'single' quotes`;
    let expected = `"all the 'single' quotes"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));

    input = `''''' fives single quotes before and after '''''`;
    expected = `"''''' fives single quotes before and after '''''"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));
  });

  it('prevents csv injection', () => {
    let input = `@@Should prepend with single quote`;
    let expected = `"'@@Should prepend with single quote"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));

    input = `at symbol @ later on, do not expect ' at start`;
    expected = `"at symbol @ later on, do not expect ' at start"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));

    input = `==@+=--@Should prepend with single quote`;
    expected = `"'==@+=--@Should prepend with single quote"`;
    assert.equal(expected, preventCSVInjectionAndStringify(input));
  });
});

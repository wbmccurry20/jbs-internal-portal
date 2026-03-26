#!/usr/bin/env node
/**
 * xls_to_csv.js — converts a legacy .xls file to CSV output on stdout
 * using SheetJS (xlsx) which handles all BIFF/XLS variants correctly.
 *
 * Usage: node xls_to_csv.js <path-to-xls-file>
 *
 * Requires: xlsx package in ../node_modules/xlsx (relative to this script)
 */

'use strict';

const path = require('path');
const fs = require('fs');

// Resolve xlsx from node_modules relative to this script's directory
const xlsxModulePath = path.join(__dirname, '..', 'node_modules', 'xlsx');
let XLSX;
try {
  XLSX = require(xlsxModulePath);
} catch (e) {
  // Fallback: try globally installed xlsx
  try {
    XLSX = require('xlsx');
  } catch (e2) {
    process.stderr.write('ERROR: xlsx module not found at ' + xlsxModulePath + '\n');
    process.stderr.write('Run: npm install xlsx  in the jbs-internal-portal directory\n');
    process.exit(1);
  }
}

const filePath = process.argv[2];
if (!filePath) {
  process.stderr.write('Usage: node xls_to_csv.js <xls_file_path>\n');
  process.exit(1);
}

if (!fs.existsSync(filePath)) {
  process.stderr.write('ERROR: File not found: ' + filePath + '\n');
  process.exit(1);
}

try {
  // Read the XLS file - SheetJS handles all BIFF variants (BIFF2-BIFF8)
  // raw: true preserves original cell values (text stays as text, not converted)
  // cellText: true generates w (formatted text) fields
  const workbook = XLSX.readFile(filePath, {
    type: 'file',
    raw: true,
    cellText: true,
    cellDates: false,
    WTF: false,
  });

  if (!workbook.SheetNames || workbook.SheetNames.length === 0) {
    process.stderr.write('ERROR: No sheets found in XLS file\n');
    process.exit(1);
  }

  const sheetName = workbook.SheetNames[0];
  const sheet = workbook.Sheets[sheetName];

  // Convert to CSV — blankrows: false skips entirely empty rows
  const csv = XLSX.utils.sheet_to_csv(sheet, {
    FS: ',',
    RS: '\n',
    blankrows: false,
    rawNumbers: false,  // use formatted text for numbers/dates
  });

  process.stdout.write(csv);
} catch (e) {
  process.stderr.write('ERROR parsing XLS: ' + e.message + '\n');
  process.exit(1);
}

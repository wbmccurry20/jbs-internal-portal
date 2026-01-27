import { describe, it, expect } from 'vitest';

describe('JBS Utility Functions', () => {
  describe('File Validation', () => {
    const isValidExcelFile = (filename: string): boolean => {
      const validExtensions = ['.xlsx', '.xls', '.csv'];
      return validExtensions.some(ext => filename.toLowerCase().endsWith(ext));
    };

    it('validates Excel file extensions', () => {
      expect(isValidExcelFile('report.xlsx')).toBe(true);
      expect(isValidExcelFile('data.xls')).toBe(true);
      expect(isValidExcelFile('export.csv')).toBe(true);
    });

    it('rejects invalid file extensions', () => {
      expect(isValidExcelFile('document.pdf')).toBe(false);
      expect(isValidExcelFile('image.jpg')).toBe(false);
      expect(isValidExcelFile('script.exe')).toBe(false);
    });

    it('is case insensitive', () => {
      expect(isValidExcelFile('REPORT.XLSX')).toBe(true);
      expect(isValidExcelFile('Data.XLS')).toBe(true);
    });
  });

  describe('File Size Validation', () => {
    const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB

    const isValidFileSize = (sizeInBytes: number): boolean => {
      return sizeInBytes > 0 && sizeInBytes <= MAX_FILE_SIZE;
    };

    it('accepts valid file sizes', () => {
      expect(isValidFileSize(1024)).toBe(true); // 1KB
      expect(isValidFileSize(1024 * 1024)).toBe(true); // 1MB
      expect(isValidFileSize(5 * 1024 * 1024)).toBe(true); // 5MB
    });

    it('rejects files that are too large', () => {
      expect(isValidFileSize(11 * 1024 * 1024)).toBe(false); // 11MB
      expect(isValidFileSize(100 * 1024 * 1024)).toBe(false); // 100MB
    });

    it('rejects zero or negative sizes', () => {
      expect(isValidFileSize(0)).toBe(false);
      expect(isValidFileSize(-1024)).toBe(false);
    });
  });

  describe('Filename Sanitization', () => {
    const sanitizeFilename = (filename: string): string => {
      // Remove path traversal attempts and dangerous characters
      return filename
        .replace(/\.\./g, '')
        .replace(/[<>:"|?*]/g, '_')
        .replace(/\\/g, '_')
        .replace(/\//g, '_');
    };

    it('removes path traversal attempts', () => {
      expect(sanitizeFilename('../../../etc/passwd')).not.toContain('..');
      expect(sanitizeFilename('../../file.xlsx')).not.toContain('..');
    });

    it('sanitizes dangerous characters', () => {
      expect(sanitizeFilename('file<script>.xlsx')).not.toContain('<');
      expect(sanitizeFilename('file|name.xlsx')).not.toContain('|');
      expect(sanitizeFilename('file:name.xlsx')).not.toContain(':');
    });

    it('preserves valid filenames', () => {
      const validName = 'report_2024_01.xlsx';
      expect(sanitizeFilename(validName)).toBe(validName);
    });
  });

  describe('Date Range Validation', () => {
    const isValidDateRange = (startDate: Date, endDate: Date): boolean => {
      return startDate <= endDate;
    };

    it('validates correct date ranges', () => {
      const start = new Date('2024-01-01');
      const end = new Date('2024-01-31');
      expect(isValidDateRange(start, end)).toBe(true);
    });

    it('accepts same start and end date', () => {
      const date = new Date('2024-01-15');
      expect(isValidDateRange(date, date)).toBe(true);
    });

    it('rejects invalid date ranges', () => {
      const start = new Date('2024-02-01');
      const end = new Date('2024-01-01');
      expect(isValidDateRange(start, end)).toBe(false);
    });
  });

  describe('Transaction Amount Formatting', () => {
    const formatAmount = (amount: number): string => {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2,
      }).format(amount);
    };

    it('formats positive amounts', () => {
      expect(formatAmount(1234.56)).toBe('$1,234.56');
      expect(formatAmount(0.99)).toBe('$0.99');
    });

    it('formats negative amounts', () => {
      expect(formatAmount(-500.00)).toBe('-$500.00');
    });

    it('handles large amounts', () => {
      expect(formatAmount(1000000)).toBe('$1,000,000.00');
    });
  });

  describe('Reconciliation Status', () => {
    type ReconciliationStatus = 'matched' | 'unmatched' | 'void' | 'ambiguous';

    const getStatusBadgeColor = (status: ReconciliationStatus): string => {
      const colors: Record<ReconciliationStatus, string> = {
        matched: 'green',
        unmatched: 'yellow',
        void: 'gray',
        ambiguous: 'orange',
      };
      return colors[status];
    };

    it('returns correct colors for each status', () => {
      expect(getStatusBadgeColor('matched')).toBe('green');
      expect(getStatusBadgeColor('unmatched')).toBe('yellow');
      expect(getStatusBadgeColor('void')).toBe('gray');
      expect(getStatusBadgeColor('ambiguous')).toBe('orange');
    });
  });

  describe('Email Validation', () => {
    const isValidEmail = (email: string): boolean => {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      return emailRegex.test(email);
    };

    it('validates correct email formats', () => {
      expect(isValidEmail('user@jbsconstructiongroup.com')).toBe(true);
      expect(isValidEmail('emily.simpson@jbs.com')).toBe(true);
    });

    it('rejects invalid emails', () => {
      expect(isValidEmail('notanemail')).toBe(false);
      expect(isValidEmail('@jbs.com')).toBe(false);
      expect(isValidEmail('user@')).toBe(false);
    });
  });

  describe('Tolerance Calculation', () => {
    const isWithinTolerance = (
      amount1: number,
      amount2: number,
      tolerancePercent: number
    ): boolean => {
      const difference = Math.abs(amount1 - amount2);
      const tolerance = Math.abs(amount1) * (tolerancePercent / 100);
      return difference <= tolerance;
    };

    it('detects amounts within tolerance', () => {
      expect(isWithinTolerance(1000, 1005, 1)).toBe(true); // 0.5% diff, 1% tolerance
      expect(isWithinTolerance(1000, 1010, 1)).toBe(true); // 1% diff, 1% tolerance
    });

    it('detects amounts outside tolerance', () => {
      expect(isWithinTolerance(1000, 1020, 1)).toBe(false); // 2% diff, 1% tolerance
    });

    it('handles negative amounts', () => {
      expect(isWithinTolerance(-1000, -1005, 1)).toBe(true);
    });
  });
});

describe('Data Processing', () => {
  describe('CSV Parsing', () => {
    const parseCSVLine = (line: string): string[] => {
      // Simple CSV parser (for demonstration)
      return line.split(',').map(field => field.trim());
    };

    it('parses simple CSV lines', () => {
      const line = 'John Doe, 2024-01-15, 1000.00';
      const fields = parseCSVLine(line);
      expect(fields).toEqual(['John Doe', '2024-01-15', '1000.00']);
    });

    it('handles empty fields', () => {
      const line = 'John Doe, , 1000.00';
      const fields = parseCSVLine(line);
      expect(fields).toEqual(['John Doe', '', '1000.00']);
    });
  });

  describe('Data Grouping', () => {
    interface Transaction {
      date: string;
      amount: number;
      description: string;
    }

    const groupByDate = (transactions: Transaction[]): Record<string, Transaction[]> => {
      return transactions.reduce((acc, transaction) => {
        const date = transaction.date;
        if (!acc[date]) {
          acc[date] = [];
        }
        acc[date].push(transaction);
        return acc;
      }, {} as Record<string, Transaction[]>);
    };

    it('groups transactions by date', () => {
      const transactions: Transaction[] = [
        { date: '2024-01-01', amount: 100, description: 'A' },
        { date: '2024-01-01', amount: 200, description: 'B' },
        { date: '2024-01-02', amount: 300, description: 'C' },
      ];

      const grouped = groupByDate(transactions);

      expect(grouped['2024-01-01']).toHaveLength(2);
      expect(grouped['2024-01-02']).toHaveLength(1);
    });

    it('handles empty array', () => {
      const grouped = groupByDate([]);
      expect(Object.keys(grouped)).toHaveLength(0);
    });
  });
});

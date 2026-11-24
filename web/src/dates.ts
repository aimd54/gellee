// Copyright The gellee Authors
// SPDX-License-Identifier: Apache-2.0

// Dates as a person would say them. Every function works on the text of
// a date and on the server's date, so the browser's clock and time zone
// never change what is shown.

const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

function utc(day: string): Date {
  return new Date(`${day.slice(0, 10)}T00:00:00Z`);
}

function dayAndMonth(d: Date): string {
  return `${d.getUTCDate()} ${MONTHS[d.getUTCMonth()]!}`;
}

// daysBetween counts the days from one date to a later one.
export function daysBetween(from: string, to: string): number {
  return Math.round((utc(to).getTime() - utc(from).getTime()) / 86_400_000);
}

// readerDate writes a day as "today", "yesterday", "9 November", or
// "9 November 2025" when the year is not the current one.
export function readerDate(day: string, today: string): string {
  const gap = daysBetween(day, today);
  if (Number.isNaN(gap)) return 'an unknown date';
  if (gap === 0) return 'today';
  if (gap === 1) return 'yesterday';
  const d = utc(day);
  return d.getUTCFullYear() === utc(today).getUTCFullYear() ? dayAndMonth(d) : `${dayAndMonth(d)} ${d.getUTCFullYear()}`;
}

// readerMoment writes a timestamp as its date and the clock time the
// machine that wrote it saw.
export function readerMoment(stamp: string, today: string): string {
  return `${readerDate(stamp.slice(0, 10), today)}, ${stamp.slice(11, 16)}`;
}

// longDay writes "Wednesday, 11 November".
export function longDay(day: string): string {
  const d = utc(day);
  return `${WEEKDAYS[d.getUTCDay()]!}, ${dayAndMonth(d)}`;
}

// collectedLine says when the git and usage files were last collected.
export function collectedLine(collected: string, today: string): string {
  if (collected === '') return 'not collected yet';
  const day = collected.slice(0, 10);
  return day === today ? `collected ${collected.slice(11, 16)}` : `collected ${readerDate(day, today)}`;
}

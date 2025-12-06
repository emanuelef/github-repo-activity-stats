# Hourly Stars Example

This example demonstrates the `GetRecentStarsHistoryByHour` function which provides hourly granularity for star history data.

## Features

- Get stars per hour for the last N days
- Find peak activity hours
- Compare hourly vs daily data granularity
- Identify trends at a finer time resolution

## Usage

```bash
cd example-hourly-stars
go run main.go
```

## What it does

1. **Example 1**: Fetches hourly stars for a popular repository (last 7 days)
   - Shows first and last few hours
   - Identifies the peak hour
   - Displays total stars accumulated

2. **Example 2**: Fetches hourly stars for a smaller repository (last 3 days)
   - Groups hourly data by day
   - Shows daily breakdown

3. **Example 3**: Compares daily vs hourly granularity (last 2 days)
   - Demonstrates the difference in detail level
   - Shows the top 5 most active hours

## Benefits of Hourly Data

- **Better trend detection**: Identify specific times of day when stars are gained
- **Peak activity analysis**: Find out when your project gets the most attention
- **Time zone insights**: Understand which hours are most active for your users
- **Marketing timing**: Optimize when to post updates based on activity patterns

# Monzo Enhanced Categories

## Concept

I'm after more granular and configurable categorisation of spending than what Monzo offers out of the box. I've started on a Vue.js frontend for this, and decided a test-driven Go backend would be a good project to put into practice what I've been leaning through [Learn Go With Tests](https://quii.gitbook.io/learn-go-with-tests/). The initial idea was to lean on the metadata field Monzo provides for transactions, although that idea may be retired if it ends up causing complicated duplication between Monzo's backend and what is being made here (I'm thinking counts of categories).

## Data Structure
* Adjacency list of categories like what Monzo provides e.g. accommodation, food & drink
* with subcategories for each category e.g. accommodation -> hotel/hostel/apartment
* Additional data (metadata?) attached to categories (I've yet to name this well) e.g. number of nights stayed for lodging, breakfast/lunch/dinner for food & drink

## Implementation
* Categories/Types:
  * Fields: id (string), name (string), parentID (string)
  * Methods: Add, Rename, Remove, List, Get
  * Considerations: no duplicate names, cannot remove a category used by a transaction

* Additional data:
  * Fields: id (string), name (string), parentID (string), type (string), options (slice of strings)
  * Methods: Add, Rename, Remove, List, Get
  * Option methods: Add, Rename, Remove, List, Get
  * Considerations: type can be "string" or "int", option methods only available to "string" type, no duplicate names, cannot remove additional data used by a transaction

## TBD
* May need counters of all types & metadata if the Monzo API is not fast enough to grab all transactions on the fly for aggregation. At this point, should the Monzo API even be used? These counters needn't know about the data structure hierarchy, can just be a list of IDs with counts.
* Support multiple users. Right now I'm ignoring this aspect.
* Request from Claire: tag every clearly superfluous purchase as "treat yo self"

# Modify a feature

## Overview

Adding or changing the functionality of a feature flow

## Steps

1. **Define requirements**
   - Clarify feature scope and goals
   - Identify user stories and acceptance criteria
   - Plan technical approach

2. **Plan architecture changes if neccessary**
   - Think on the data flow according to our documentation
   - Design data models and APIs
   - Think on the data layers and files to be modified

3. **Modify the tests for the feature changes to implement**
   - Using the testing rules setted up, update or create the necesary tests to comply with the feature requierements

4. **Write the code**
   - Write/Update the necesary files according our rules and skills to make tests pass.

5. **Verify all the tests pass**
   - Make sure all the tests pass

6. **Update necesary documentation**
   -Update our feature documentation

## Feature Setup Checklist

- [ ] Requirements documented
- [ ] Technical approach planned
- [ ] Development environment ready
- [ ] All tests pass
- [ ] New API/handlers documented with swaggo; `make swag` run if needed

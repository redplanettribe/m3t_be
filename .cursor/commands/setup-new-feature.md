# Setup New Feature

## Overview

Systematically set up a new feature from initial planning through to implementation structure.

## Steps

1. **Define requirements**
   - Clarify feature scope and goals
   - Identify user stories and acceptance criteria
   - Plan technical approach

2. **Create feature branch**
   - Branch from `main`
   - Set up local development environment
   - Configure any new dependencies

3. **Plan architecture**
   - Think on the data flow according to our documentation
   - Design data models and APIs
   - Think on the data layers and files to be modified

4. **Write the tests for the feature to implement**
   - Using the testing rules setted up, create the necesary tests to comply with the feature requierements

5. **Write the code**
   - Write/Update the necesary files according our rules and skills to make tests pass.

6. **Verify all the tests pass**
   - Make sure all the tests pass
7. **Update necesary documentation**
   -Update our feature documentation

## Feature Setup Checklist

- [ ] Requirements documented
- [ ] User stories written
- [ ] Technical approach planned
- [ ] Feature branch created
- [ ] Development environment ready
- [ ] All tests pass
- [ ] New API/handlers documented with swaggo; `make swag` run if needed

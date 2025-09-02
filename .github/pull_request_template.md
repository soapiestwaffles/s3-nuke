## Description

Brief description of what this PR accomplishes.

Fixes # (issue)

## Type of Change

Please delete options that are not relevant.

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Refactoring (no functional changes)
- [ ] Tests (adding or updating tests)
- [ ] CI/CD changes

## Testing

Please describe the tests that you ran to verify your changes.

- [ ] Unit tests pass (`make test`)
- [ ] Linting passes (`make lint-go`)
- [ ] Manual testing completed
- [ ] Tested with real AWS S3 buckets (if applicable)
- [ ] Tested edge cases and error scenarios

**Test Configuration:**
- OS: 
- Go version: 
- AWS region (if applicable): 

## Security Considerations

Since s3-nuke performs destructive operations on AWS resources:

- [ ] No hardcoded credentials or sensitive information
- [ ] Proper error handling for AWS API calls
- [ ] Safety checks and confirmations maintained
- [ ] No unintended side effects on AWS resources

## Performance Impact

- [ ] No performance degradation
- [ ] Performance improvement (describe below)
- [ ] Performance impact assessed and acceptable

## Documentation

- [ ] Code is self-documenting with clear variable/function names
- [ ] Complex logic includes comments
- [ ] README.md updated (if needed)
- [ ] Help text/usage updated (if needed)

## Breaking Changes

If this introduces breaking changes, please describe:
- What breaks
- Migration path for users
- Version bump required

## Checklist

- [ ] My code follows the existing code style
- [ ] I have performed a self-review of my own code
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings or errors
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

## Additional Notes

Any additional information that reviewers should know about this PR.
/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
)

func (checker *Checker) VisitSwitchStatement(statement *ast.SwitchStatement) ast.Repr {

	testType := checker.VisitExpression(statement.Expression, nil)

	testTypeIsValid := !testType.IsInvalidType()

	// The test expression must be equatable

	if testTypeIsValid && !testType.IsEquatable() {
		checker.report(
			&NotEquatableTypeError{
				Type:  testType,
				Range: ast.NewRangeFromPositioned(statement.Expression),
			},
		)
	}

	// Check all cases

	caseCount := len(statement.Cases)

	for i, switchCase := range statement.Cases {
		// Only one default case is allowed, as the last case
		defaultAllowed := i == caseCount-1
		checker.visitSwitchCase(switchCase, defaultAllowed, testType, testTypeIsValid)
	}

	checker.checkDuplicateCases(statement.Cases)

	checker.functionActivations.WithSwitch(func() {
		checker.checkSwitchCasesStatements(statement.Cases)
	})

	return nil
}

func (checker *Checker) visitSwitchCase(
	switchCase *ast.SwitchCase,
	defaultAllowed bool,
	testType Type,
	testTypeIsValid bool,
) {
	caseExpression := switchCase.Expression

	// If the case has no expression, it is a default case

	if caseExpression == nil {

		// Only one default case is allowed, as the last case
		if !defaultAllowed {
			checker.report(
				&SwitchDefaultPositionError{
					Range: switchCase.Range,
				},
			)
		}
	} else {
		checker.checkSwitchCaseExpression(caseExpression, testType, testTypeIsValid)
	}
}

func (checker *Checker) checkSwitchCaseExpression(
	caseExpression ast.Expression,
	testType Type,
	testTypeIsValid bool,
) {

	caseType := checker.VisitExpression(caseExpression, nil)

	if caseType.IsInvalidType() {
		return
	}

	// The type of each case expression must be the same
	// as the type of the test expression

	if testTypeIsValid {
		// If the test type is valid,
		// the case type can be checked to be equatable and compatible in one go

		if !AreCompatibleEquatableTypes(testType, caseType) {
			checker.report(
				&InvalidBinaryOperandsError{
					Operation: ast.OperationEqual,
					LeftType:  testType,
					RightType: caseType,
					Range:     ast.NewRangeFromPositioned(caseExpression),
				},
			)
		}
	} else {
		// If the test type is invalid,
		// at least the case type can be checked to be equatable

		if !caseType.IsEquatable() {
			checker.report(
				&NotEquatableTypeError{
					Type:  caseType,
					Range: ast.NewRangeFromPositioned(caseExpression),
				},
			)
		}
	}
}

func (checker *Checker) checkSwitchCasesStatements(cases []*ast.SwitchCase) {
	caseCount := len(cases)
	if caseCount == 0 {
		return
	}

	// NOTE: always check blocks as if they're only *potentially* evaluated.
	// However, the default case's block must be checked directly as the "else",
	// because if a default case exists, the whole switch statement
	// will definitely have one case which will be taken.

	switchCase := cases[0]

	if caseCount == 1 {
		if switchCase.Expression == nil {
			checker.checkSwitchCaseStatements(switchCase)
			return
		}
	}

	_, _ = checker.checkConditionalBranches(
		func() Type {
			checker.checkSwitchCaseStatements(switchCase)
			return nil
		},
		func() Type {
			checker.checkSwitchCasesStatements(cases[1:])
			return nil
		},
	)
}

func (checker *Checker) checkSwitchCaseStatements(switchCase *ast.SwitchCase) {

	// Switch-cases must have at least one statement.
	// This avoids cases that look like implicit fallthrough is assumed.

	if len(switchCase.Statements) == 0 {
		checker.report(
			&MissingSwitchCaseStatementsError{
				Pos: switchCase.EndPosition().Shifted(1),
			},
		)
		return
	}

	// NOTE: the block ensures that the statements are checked in a new scope

	block := &ast.Block{
		Statements: switchCase.Statements,
		Range: ast.Range{
			StartPos: switchCase.Statements[0].StartPosition(),
			EndPos:   switchCase.EndPos,
		},
	}
	block.Accept(checker)
}

func (checker *Checker) checkDuplicateCases(cases []*ast.SwitchCase) {
	duplicates := make(map[*ast.SwitchCase]bool)

	duplicateChecker := &duplicateCaseChecker{}

	for i, switchCase := range cases {

		// If the current case is already identified as a duplicate,
		// the no need to check it again. Can simply skip.
		if _, isDuplicate := duplicates[switchCase]; isDuplicate {
			continue
		}

		for j := i + 1; j < len(cases); j++ {
			otherCase := cases[j]
			if duplicateChecker.isDuplicate(switchCase.Expression, otherCase.Expression) {
				duplicates[otherCase] = true

				checker.report(
					&DuplicateSwitchCaseError{
						Range: ast.NewRangeFromPositioned(otherCase.Expression),
					},
				)
			}
		}
	}
}

var _ ast.ExpressionVisitor = &duplicateCaseChecker{}

type duplicateCaseChecker struct {
	expr ast.Expression
}

func (d *duplicateCaseChecker) isDuplicate(this ast.Expression, other ast.Expression) bool {
	tempExpr := d.expr
	d.expr = this
	defer func() {
		d.expr = tempExpr
	}()

	return other.AcceptExp(d).(bool)
}

func (d *duplicateCaseChecker) VisitBoolExpression(otherExpr *ast.BoolExpression) ast.Repr {
	expr, ok := d.expr.(*ast.BoolExpression)
	if !ok {
		return false
	}

	return otherExpr.Value == expr.Value
}

func (d *duplicateCaseChecker) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	_, ok := d.expr.(*ast.NilExpression)
	return ok
}

func (d *duplicateCaseChecker) VisitIntegerExpression(otherExpr *ast.IntegerExpression) ast.Repr {
	expr, ok := d.expr.(*ast.IntegerExpression)
	if !ok {
		return false
	}

	return expr.Value.Cmp(otherExpr.Value) == 0
}

func (d *duplicateCaseChecker) VisitFixedPointExpression(otherExpr *ast.FixedPointExpression) ast.Repr {
	expr, ok := d.expr.(*ast.FixedPointExpression)
	if !ok {
		return false
	}

	return expr.Negative == otherExpr.Negative &&
		expr.Fractional.Cmp(otherExpr.Fractional) == 0 &&
		expr.UnsignedInteger.Cmp(otherExpr.UnsignedInteger) == 0 &&
		expr.Scale == otherExpr.Scale
}

func (d *duplicateCaseChecker) VisitArrayExpression(otherExpr *ast.ArrayExpression) ast.Repr {
	expr, ok := d.expr.(*ast.ArrayExpression)
	if !ok || len(expr.Values) != len(otherExpr.Values) {
		return false
	}

	for index, value := range expr.Values {
		if !d.isDuplicate(value, otherExpr.Values[index]) {
			return false
		}
	}

	return true
}

func (d *duplicateCaseChecker) VisitDictionaryExpression(otherExpr *ast.DictionaryExpression) ast.Repr {
	expr, ok := d.expr.(*ast.DictionaryExpression)
	if !ok || len(expr.Entries) != len(otherExpr.Entries) {
		return false
	}

	for index, entry := range expr.Entries {
		otherEntry := otherExpr.Entries[index]

		if !d.isDuplicate(entry.Key, otherEntry.Key) ||
			!d.isDuplicate(entry.Value, otherEntry.Value) {
			return false
		}
	}

	return true
}

func (d *duplicateCaseChecker) VisitIdentifierExpression(otherExpr *ast.IdentifierExpression) ast.Repr {
	expr, ok := d.expr.(*ast.IdentifierExpression)
	if !ok {
		return false
	}

	return expr.Identifier == otherExpr.Identifier
}

func (d *duplicateCaseChecker) VisitInvocationExpression(otherExpr *ast.InvocationExpression) ast.Repr {
	// Invocations can be stateful. Thus, it's not possible to determine if
	// calling the same function in two cases would produce the same results.
	return false
}

func (d *duplicateCaseChecker) VisitMemberExpression(otherExpr *ast.MemberExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitIndexExpression(otherExpr *ast.IndexExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitConditionalExpression(otherExpr *ast.ConditionalExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitUnaryExpression(otherExpr *ast.UnaryExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitBinaryExpression(otherExpr *ast.BinaryExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitFunctionExpression(otherExpr *ast.FunctionExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitStringExpression(otherExpr *ast.StringExpression) ast.Repr {
	expr, ok := d.expr.(*ast.StringExpression)
	if !ok {
		return false
	}

	return expr.Value == otherExpr.Value
}

func (d *duplicateCaseChecker) VisitCastingExpression(otherExpr *ast.CastingExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitCreateExpression(otherExpr *ast.CreateExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitDestroyExpression(otherExpr *ast.DestroyExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitReferenceExpression(otherExpr *ast.ReferenceExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitForceExpression(otherExpr *ast.ForceExpression) ast.Repr {
	panic("implement me")
}

func (d *duplicateCaseChecker) VisitPathExpression(otherExpr *ast.PathExpression) ast.Repr {
	panic("implement me")
}
